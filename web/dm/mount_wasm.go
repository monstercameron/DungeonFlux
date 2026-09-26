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
	ui.UseEffect(func() func() {
		ctx, cancel := context.WithCancel(context.Background())
		token := dmToken()
		updates := props.client.watch(ctx, token)
		go func() {
			for next := range updates {
				state.Set(next)
			}
		}()
		if props.client.audio != nil {
			listen := audio.NewListenClient(listenServiceAdapter{client: props.client.audio}).Listen(ctx, token)
			player := audio.NewPlayer()
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
			_ = props.client.close()
		}
	})
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
		return func() {
			if player != nil {
				player.Stop()
			}
		}
	}, music)
	unlock := ui.UseEvent(func() {
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
			children = append(children, LobbyComponent(NewLobbyModel(roomCode, ""))(router.Attrs{}))
		case LayerScene:
			children = append(children, SceneComponent(view)(router.Attrs{}))
		case LayerCallout:
			children = append(children, CalloutComponent(CalloutViewFromDMView(view))(router.Attrs{}))
		case LayerClip:
			children = append(children, ClipComponent(ClipModelFromView(view))(router.Attrs{}))
		case LayerDice:
			children = append(children, DiceComponent(DiceViewFromDMView(view))(router.Attrs{}))
		case LayerTimer:
			children = append(children, TimerComponent(TimerViewFromDMView(view))(router.Attrs{}))
		case LayerCombat:
			children = append(children, CombatComponent(view)(router.Attrs{}))
		case LayerEnd:
			children = append(children, EndCardComponent(NewEndCardModel())(router.Attrs{}))
		}
	}
	children = append(children, html.Button(html.Props{Type: "button", Class: "df-dm-audio-unlock", OnClick: unlock}, html.Text("Enable table audio")))
	return html.Main(html.Props{Class: "df-dm-screen", Role: "main"}, children...)
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
