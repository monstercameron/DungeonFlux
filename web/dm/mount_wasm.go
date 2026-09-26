//go:build js && wasm

package dm

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"syscall/js"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/shell/audio"
	"github.com/monstercameron/GoGRPCBridge/pkg/wasm/dialer"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"google.golang.org/grpc"
)

type watchClient interface {
	Watch(context.Context, *dungeonfluxv1.WatchRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[dungeonfluxv1.WatchMessage], error)
}

type screenClient struct {
	conn  *grpc.ClientConn
	api   watchClient
	audio interface {
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
	conn, err := grpc.NewClient(endpoint, dialer.New(endpoint))
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
		stream, err := c.api.Watch(ctx, &dungeonfluxv1.WatchRequest{SeatToken: token})
		if err != nil {
			return
		}
		for {
			message, recvErr := stream.Recv()
			if recvErr != nil {
				return
			}
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

// Mount returns the stateful DM screen for the shared shell router.
func Mount(endpoint string) router.Component {
	return func(_ router.Attrs) *router.Element {
		client, err := newScreenClient(endpoint)
		if err != nil {
			return ui.CreateElement(screenError, err.Error())
		}
		return ui.CreateElement(screenView, screenProps{client: client})
	}
}

type screenProps struct{ client *screenClient }

func screenError(message string) ui.Node {
	return html.Main(html.Props{Class: "df-dm-error", Role: "main"}, html.H1(html.Props{}, html.Text("DungeonFlux")), html.P(html.Props{Role: "alert"}, html.Text(message)))
}

func screenView(props screenProps) ui.Node {
	state := ui.UseState((*dungeonfluxv1.ScreenState)(nil))
	musicPlayer := ui.UseState((*MusicPlayer)(nil))
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
			listen := audio.NewListenClient(listenServiceAdapter{client: props.client.audio}).Listen(ctx, token)
			player := voicePlayer.Get()
			if player == nil {
				player = audio.NewPlayer()
				voicePlayer.Set(player)
			}
			go func() {
				adapter := NewListenAudio(player)
				for result := range listen {
					if result.Message != nil {
						_ = adapter.Handle(result.Message)
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
			_ = props.client.close()
		}
	}, props.client, token)
	snapshot := state.Get()
	music := MusicModelFromView(snapshot.GetDm())
	ui.UseEffect(func() func() {
		player := musicPlayer.Get()
		if player == nil {
			player = NewMusicPlayer()
			musicPlayer.Set(player)
		}
		if player != nil {
			_ = player.Apply(music, 0)
		}
		return nil
	}, music)
	ui.UseEffect(func() func() {
		return func() {
			if player := musicPlayer.Get(); player != nil {
				player.Close()
			}
		}
	}, props.client)
	unlock := ui.UseEvent(func() {
		if player := voicePlayer.Get(); player != nil {
			_ = player.Resume()
		}
		if player := musicPlayer.Get(); player != nil {
			_ = player.Resume()
		}
	})
	return compose(snapshot, dmRoomCode(), unlock)
}

func dmToken() string {
	location := js.Global().Get("location")
	if !location.Truthy() {
		return ""
	}
	params := js.Global().Get("URLSearchParams").New(location.Get("search"))
	value := params.Call("get", "token")
	if value.IsNull() || value.IsUndefined() {
		return ""
	}
	return value.String()
}

func compose(state *dungeonfluxv1.ScreenState, roomCode string, unlock ui.Handler) ui.Node {
	view := state.GetDm()
	layers := SelectLayers(state)
	children := make([]ui.Node, 0, len(layers))
	for _, layer := range layers {
		switch layer {
		case LayerLobby:
			children = appendLayer(children, layer, LobbyComponent(NewLobbyModel(roomCode, ""))(router.Attrs{}))
		case LayerScene:
			children = appendLayer(children, layer, SceneComponent(view)(router.Attrs{}))
		case LayerCallout:
			children = appendLayer(children, layer, CalloutComponent(CalloutViewFromDMView(view))(router.Attrs{}))
		case LayerClip:
			children = appendLayer(children, layer, ClipComponent(ClipModelFromView(view))(router.Attrs{}))
		case LayerDice:
			children = appendLayer(children, layer, DiceComponent(DiceViewFromDMView(view))(router.Attrs{}))
		case LayerTimer:
			children = appendLayer(children, layer, TimerComponent(TimerViewFromDMView(view))(router.Attrs{}))
		case LayerCombat:
			children = appendLayer(children, layer, CombatComponent(view)(router.Attrs{}))
		case LayerEnd:
			children = appendLayer(children, layer, EndCardComponent(NewEndCardModel())(router.Attrs{}))
		}
	}
	children = append(children, html.Button(html.Props{Type: "button", Class: "df-dm-audio-unlock", OnClick: unlock, Style: map[string]string{"position": "absolute", "right": "1rem", "top": "1rem", "z-index": "100"}}, html.Text("Enable table audio")))
	stage := html.Div(html.Props{Class: "df-dm-stage", Style: map[string]string{"position": "relative", "width": "100%", "aspect-ratio": "16 / 9", "overflow": "hidden"}}, children...)
	return html.Main(html.Props{Class: "df-dm-screen", Role: "main", Style: map[string]string{"width": "100%", "max-width": "1920px", "margin": "0 auto", "padding": "0", "box-sizing": "border-box"}}, stage)
}

func appendLayer(children []ui.Node, layer Layer, content ui.Node) []ui.Node {
	return append(children, html.Div(html.Props{Class: "df-dm-layer df-dm-layer-" + string(layer), Style: layerStyle(layer)}, content))
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
