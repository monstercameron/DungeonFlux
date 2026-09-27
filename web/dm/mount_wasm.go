//go:build js && wasm

package dm

import (
	"context"
	"errors"
	"google.golang.org/grpc/backoff"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"syscall/js"
	"time"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/shell/audio"
	"github.com/monstercameron/DungeonFlux/web/shell/watch"
	"github.com/monstercameron/GoGRPCBridge/pkg/wasm/dialer"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type watchClient interface {
	Watch(context.Context, *dungeonfluxv1.WatchRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[dungeonfluxv1.WatchMessage], error)
}

type screenClient struct {
	recovery watch.Controller
	conn     *grpc.ClientConn
	api      watchClient
	audio    interface {
		Listen(context.Context, *dungeonfluxv1.ListenRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[dungeonfluxv1.AudioMessage], error)
	}
}

type listenServiceAdapter struct {
	client interface {
		Listen(context.Context, *dungeonfluxv1.ListenRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[dungeonfluxv1.AudioMessage], error)
	}
}

func (a listenServiceAdapter) Listen(ctx context.Context, request *dungeonfluxv1.ListenRequest) (audio.Stream, error) {
	return a.client.Listen(ctx, request)
}

func newScreenClient(endpoint string) (*screenClient, error) {
	endpoint = browserEndpoint(endpoint)
	if endpoint == "" {
		return nil, errors.New("dm: endpoint is required")
	}
	conn, err := grpc.NewClient("passthrough:///dungeonflux", dialer.New(endpoint), grpc.WithTransportCredentials(insecure.NewCredentials()),
		// Cap reconnect backoff (gRPC defaults to 120 s) so a dropped socket recovers fast.
		grpc.WithConnectParams(grpc.ConnectParams{Backoff: backoff.Config{BaseDelay: 250 * time.Millisecond, Multiplier: 1.6, Jitter: 0.2, MaxDelay: 3 * time.Second}, MinConnectTimeout: 5 * time.Second}))
	if err != nil {
		return nil, err
	}
	return &screenClient{conn: conn, api: dungeonfluxv1.NewSessionServiceClient(conn), audio: dungeonfluxv1.NewAudioServiceClient(conn)}, nil
}

func (c *screenClient) close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *screenClient) watch(ctx context.Context, token string) <-chan *dungeonfluxv1.ScreenState {
	states := make(chan *dungeonfluxv1.ScreenState, 1)
	go func() {
		defer close(states)
		if c == nil || c.api == nil {
			return
		}
		for message := range c.recovery.Messages(ctx, c.api, token) {
			if state := message.GetState(); state != nil {
				select {
				case states <- state:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return states
}

// sharedScreenClients keeps one gRPC connection per endpoint for the page's
// lifetime. The router re-renders the route whenever art finishes loading, so
// creating the client inside the route function reconnected (and dropped the
// Watch stream) dozens of times during start-up.
var sharedScreenClients = struct {
	sync.Mutex
	byEndpoint map[string]*screenClient
}{byEndpoint: map[string]*screenClient{}}

func sharedScreenClient(endpoint string) (*screenClient, error) {
	sharedScreenClients.Lock()
	defer sharedScreenClients.Unlock()
	if client, ok := sharedScreenClients.byEndpoint[endpoint]; ok {
		return client, nil
	}
	client, err := newScreenClient(endpoint)
	if err != nil {
		return nil, err
	}
	watch.InstallTriggers(&client.recovery)
	sharedScreenClients.byEndpoint[endpoint] = client
	return client, nil
}

// routeRenders changes on every route render. The shell re-navigates when art
// finishes loading; with the shared client the props would otherwise be equal
// and the screen would skip the render that picks up the new Blob URLs.
var routeRenders atomic.Uint64

// Mount returns the stateful DM screen for the shared shell router.
func Mount(endpoint string) router.Component {
	return func(_ router.Attrs) *router.Element {
		client, err := sharedScreenClient(endpoint)
		if err != nil {
			return ui.CreateElement(screenError, err.Error())
		}
		return ui.CreateElement(screenView, screenProps{client: client, render: routeRenders.Add(1)})
	}
}

type screenProps struct {
	client *screenClient
	render uint64
}

func screenError(message string) ui.Node {
	return html.Main(html.Props{Class: "df-dm-error", Role: "main"}, html.H1(html.Props{}, html.Text(ErrorTitle("en"))), html.P(html.Props{Role: "alert"}, html.Text(message)))
}

func reconnectingListen(ctx context.Context, service audio.Service, token string) <-chan audio.Result {
	results := make(chan audio.Result, 16)
	go func() {
		defer close(results)
		delay := 250 * time.Millisecond
		for ctx.Err() == nil {
			stream := audio.NewListenClient(service).WithDebug(audio.DebugConsole()).Listen(ctx, token)
			received := false
			for result := range stream {
				if result.Err != nil {
					break
				}
				received = true
				select {
				case results <- result:
				case <-ctx.Done():
					return
				}
			}
			if ctx.Err() != nil {
				return
			}
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return
			}
			delay = min(delay*2, 3*time.Second)
			if received {
				delay = 250 * time.Millisecond
			}
		}
	}()
	return results
}

func screenView(props screenProps) ui.Node {
	state := ui.UseState((*dungeonfluxv1.ScreenState)(nil))
	voicePlayer := ui.UseState((*audio.Player)(nil))
	token := dmToken()
	ui.UseEffect(func() func() {
		ctx, cancel := context.WithCancel(context.Background())
		updates := props.client.watch(ctx, token)
		go func() {
			for next := range updates {
				state.Set(next)
			}
		}()
		if props.client.audio != nil {
			listen := reconnectingListen(ctx, listenServiceAdapter{client: props.client.audio}, token)
			player := voicePlayer.Get()
			if player == nil {
				player = audio.NewPlayer()
				voicePlayer.Set(player)
			}
			go func() {
				for result := range listen {
					if result.Message != nil {
						_ = player.Handle(result.Message)
					}
				}
			}()
		}
		return func() {
			cancel()
			if player := voicePlayer.Get(); player != nil {
				player.Cancel("")
				player.Close()
			}
		}
	}, props.client, token)
	snapshot := state.Get()
	ui.UseEffect(func() func() {
		if player := voicePlayer.Get(); player != nil {
			_ = PlayLobbyAudio(player, strings.ToLower(strings.TrimSpace(snapshot.GetPhase())))
		}
		return nil
	}, voicePlayer.Get(), snapshot.GetPhase())
	audioOn := ui.UseState(false)
	unlock := ui.UseEvent(func() {
		if player := voicePlayer.Get(); player != nil {
			_ = ResumeAudio(player)
			_ = PlayLobbyAudio(player, strings.ToLower(strings.TrimSpace(snapshot.GetPhase())))
		}
		tableAudioUnlocked = true
		audioOn.Set(true)
	})
	useTransitionClock(snapshot.GetPhase())
	return compose(snapshot, dmRoomCode(), unlock)
}

func dmToken() string {
	location := js.Global().Get("location")
	if !location.Truthy() {
		return ""
	}
	params := js.Global().Get("URLSearchParams").New(location.Get("search"))
	value := params.Call("get", "token")
	if value.IsNull() || value.IsUndefined() || value.String() == "" {
		// The shell saves the boot token before the router drops the query.
		if storage := js.Global().Get("sessionStorage"); storage.Truthy() {
			if saved := storage.Call("getItem", "df-dm-token"); saved.Truthy() {
				return saved.String()
			}
		}
		return ""
	}
	return value.String()
}

// tableAudioUnlocked hides the "Enable table audio" chip once the table has
// tapped it: the chip sat over every TV screen, including the combat title and
// the end card, for the whole show.
var tableAudioUnlocked bool

func compose(state *dungeonfluxv1.ScreenState, roomCode string, unlock ui.Handler, preview ...bool) ui.Node {
	now := transitionNowMS()
	tvTransitions.Observe(state, now)
	tx, active := tvTransitions.Active(now)
	outgoing := tvTransitions.Outgoing(now)
	locale := localeOrDefault(state.GetDm().GetLocale())
	children := []ui.Node{themeStyles()}
	if outgoing != nil {
		// The previous phase keeps its layer keys, so the reconciler reuses its
		// DOM (and any running animation or video) under the exit class.
		children = append(children, phaseLayers(outgoing, roomCode, outgoingLayerClass(tx))...)
	}
	children = append(children, phaseLayers(state, roomCode, "")...)
	children = append(children, transitionVeil(tx, active, state, tvTransitions.Seq())...)
	if len(preview) > 0 && preview[0] {
		children = append(children, html.Div(html.Props{Role: "status", Style: map[string]string{"position": "absolute", "left": "28px", "bottom": "24px", "padding": "8px 14px", "font-size": "16px", "color": "#efe6d2", "background": "rgba(8,12,18,.9)", "border": "1px solid #8d7548", "border-radius": "6px", "z-index": "100"}}, html.Text(T(locale, "dm.preview.silent", nil))))
	} else if !tableAudioUnlocked {
		children = append(children, html.Button(html.Props{Type: "button", Class: "df-dm-audio-unlock", OnClick: unlock, Style: map[string]string{"position": "absolute", "right": "1rem", "bottom": "1rem", "z-index": "100"}}, html.Text(AudioUnlock(locale))))
	}
	stage := html.Div(html.Props{Class: "df-dm-stage"}, children...)
	canvas := html.Div(html.Props{Class: "df-dm-canvas"}, stage)
	coverState := state
	if outgoing != nil {
		coverState = outgoing
	}
	cover := html.Div(html.Props{Class: "df-dm-cover", Aria: map[string]string{"hidden": "true"}, Style: coverBackgroundStyle(coverState)})
	screen := html.Props{Class: "df-dm-screen " + currentAspectClass() + coverArtClass(coverState) + transitionScreenClass(tx, active), Role: "main"}
	// The lobby's storm sits between its title art and its text. Other phases
	// keep an empty slot so the canvas never shifts position and remounts.
	weather := html.Div(html.Props{Key: "df-wx-off", Hidden: true})
	switch strings.ToLower(strings.TrimSpace(coverState.GetPhase())) {
	case "", "lobby":
		weather = lobbyWeather()
	}
	return html.Main(screen, cover, weather, canvas)
}

// coverArtClass marks phases whose backdrop is plain art (lobby, creation,
// end): the cover alone paints it, unblurred, and the stage layer drops its
// own copy, so the whole window is one continuous background. Scene and
// combat phases keep their stage-aligned art and a blurred cover.
func coverArtClass(state *dungeonfluxv1.ScreenState) string {
	switch strings.ToLower(strings.TrimSpace(state.GetPhase())) {
	case "", "lobby", "creation", "end":
		return " df-cover-art"
	}
	return ""
}

// coverBackgroundStyle paints the area outside the 16:9 stage with the same
// art the stage shows for the phase (blurred and dimmed by .df-dm-cover), so
// the screen reads as one background. It used the tavern scene for every
// phase after the lobby, so the end card, creation and combat showed a
// second, unrelated image in the letterbox bands.
func coverBackgroundStyle(state *dungeonfluxv1.ScreenState) map[string]string {
	background := titleArtFor(currentAspectClass(), ArtURL).Background
	switch phase := strings.ToLower(strings.TrimSpace(state.GetPhase())); phase {
	case "", "lobby":
	case "creation":
		background = ArtURL("ui/title_bg_wide")
	case "end":
		background = ArtURL("ui/end_bg")
	case "combat", "cliffhanger":
		// The splat and the clips fill the stage; a dark field extends them.
		background = ""
	default:
		background = sceneBackgroundURL(state.GetDm())
	}
	if background == "" {
		background = "linear-gradient(135deg,#13283c,#0f1117 70%)"
	} else {
		background = "linear-gradient(180deg,rgba(7,11,18,.28),rgba(7,10,15,.68)),url('" + background + "')"
	}
	return map[string]string{"background-image": background, "background-size": "cover", "background-position": "center"}
}

// appendPhaseLayer keys each layer by layer and phase so a phase change mounts
// fresh DOM. The reconciler reuses nodes and leaves inline style properties
// that the new style map omits, so without the key the opening's panels kept
// the previous screen's borders, widths and offsets.
func appendPhaseLayer(children []ui.Node, layer Layer, phase, extra string, content ui.Node) []ui.Node {
	children = append(children, html.Div(html.Props{Class: "df-dm-layer df-dm-layer-" + string(layer) + extra, Style: layerStyle(layer)}, content))
	children[len(children)-1] = html.WithKey(children[len(children)-1], string(layer)+":"+phase)
	return children
}

func layerStyle(layer Layer) map[string]string {
	style := map[string]string{"position": "absolute", "width": "100%", "height": "100%", "pointer-events": "none", "z-index": layerZIndex(layer)}
	switch layer {
	case LayerDice:
		style["inset"] = "18% 0 auto"
		style["width"], style["height"] = "auto", "auto"
	case LayerTimer:
		style["inset"] = "auto 2rem 2rem auto"
		style["width"], style["height"] = "auto", "auto"
	default:
		style["inset"] = "0"
	}
	return style
}

func layerZIndex(layer Layer) string {
	switch layer {
	case LayerCallout:
		return "30"
	case LayerDice, LayerTimer:
		return "40"
	case LayerCombat:
		return "20"
	case LayerClip:
		return "20"
	case LayerEnd:
		return "50"
	case LayerScene:
		return "10"
	case LayerHUD:
		return "15"
	default:
		return "1"
	}
}

func browserEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	if parsed.IsAbs() {
		switch parsed.Scheme {
		case "http":
			parsed.Scheme = "ws"
		case "https":
			parsed.Scheme = "wss"
		case "ws", "wss":
		default:
			return ""
		}
		return parsed.String()
	}
	location := js.Global().Get("location")
	if !location.Truthy() {
		return endpoint
	}
	origin := location.Get("origin").String()
	parsed, err = url.Parse(origin + endpoint)
	if err != nil {
		return ""
	}
	if parsed.Scheme == "http" {
		parsed.Scheme = "ws"
	} else if parsed.Scheme == "https" {
		parsed.Scheme = "wss"
	}
	return parsed.String()
}

func dmRoomCode() string {
	value := browserQuery("room")
	return strings.ToUpper(strings.TrimSpace(value))
}

func browserQuery(key string) string {
	location := js.Global().Get("location")
	if !location.Truthy() {
		return ""
	}
	params := js.Global().Get("URLSearchParams").New(location.Get("search"))
	value := params.Call("get", key)
	if value.IsNull() || value.IsUndefined() {
		return ""
	}
	return value.String()
}
