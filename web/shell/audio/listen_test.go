package audio

import (
	"context"
	"errors"
	"io"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc/metadata"
)

func TestListenClient_StreamsMessagesAndSendsToken(t *testing.T) {
	fake := &fakeService{stream: &fakeStream{messages: []*dungeonfluxv1.AudioMessage{{Message: &dungeonfluxv1.AudioMessage_Frame{Frame: &dungeonfluxv1.AudioFrame{UtteranceId: "line"}}}}}}
	results := NewListenClient(fake).Listen(context.Background(), "dm-token")
	got := <-results
	if got.Err != nil || got.Message.GetFrame().GetUtteranceId() != "line" {
		t.Fatalf("result = %#v", got)
	}
	if got := <-results; !errors.Is(got.Err, io.EOF) {
		t.Fatalf("terminal error = %v", got.Err)
	}
	if fake.token != "dm-token" {
		t.Fatalf("token = %q", fake.token)
	}
}

func TestListenClient_ReportsOpenError(t *testing.T) {
	want := errors.New("open failed")
	results := NewListenClient(&fakeService{err: want}).Listen(context.Background(), "")
	if got := <-results; !errors.Is(got.Err, want) {
		t.Fatalf("error = %v", got.Err)
	}
}

type fakeService struct {
	stream Stream
	err    error
	token  string
}

func (f *fakeService) Listen(_ context.Context, request *dungeonfluxv1.ListenRequest) (Stream, error) {
	f.token = request.GetSeatToken()
	return f.stream, f.err
}

type fakeStream struct {
	messages []*dungeonfluxv1.AudioMessage
}

func (f *fakeStream) Recv() (*dungeonfluxv1.AudioMessage, error) {
	if len(f.messages) == 0 {
		return nil, io.EOF
	}
	message := f.messages[0]
	f.messages = f.messages[1:]
	return message, nil
}

func (f *fakeStream) Header() (metadata.MD, error) { return nil, nil }
func (f *fakeStream) Trailer() metadata.MD         { return nil }
func (f *fakeStream) CloseSend() error             { return nil }
func (f *fakeStream) Context() context.Context     { return context.Background() }
func (f *fakeStream) SendMsg(any) error            { return nil }
func (f *fakeStream) RecvMsg(any) error            { return io.EOF }
