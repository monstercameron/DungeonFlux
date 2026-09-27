package ports

import "context"

// ReferenceVideoRequest asks for a clip conditioned on reference images
// instead of a first frame (Seedance reference-to-video: the prompt names the
// images as @Image1, @Image2, ... in References order).
type ReferenceVideoRequest struct {
	Meta       CallMeta
	Prompt     string
	References [][]byte
	Seconds    int
	Resolution string
	Aspect     string
}

// ReferenceVideoGen submits reference-conditioned clips to a queue-based
// video vendor, polls them, and downloads the finished file.
type ReferenceVideoGen interface {
	SubmitReference(context.Context, ReferenceVideoRequest) (VideoJob, error)
	Poll(context.Context, VideoJob) (VideoStatus, error)
	Download(context.Context, string) ([]byte, error)
}
