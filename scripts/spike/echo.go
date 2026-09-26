package main

import (
	"io"

	spikev1 "github.com/monstercameron/DungeonFlux/scripts/spike/gen"
)

// echoServer sends every received audio chunk back to the caller unchanged.
// Keeping sequence and PCM bytes intact exercises the tunnel's streaming path.
type echoServer struct {
	spikev1.UnimplementedEchoServer
}

func (echoServer) Stream(stream spikev1.Echo_StreamServer) error {
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := stream.Send(chunk); err != nil {
			return err
		}
		if chunk.GetFinal() {
			return nil
		}
	}
}
