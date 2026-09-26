package openai

import (
	"testing"
)

func TestReferenceRequest_UsesOpaqueTurnaroundCanvas(t *testing.T) {
	request := ReferenceRequest("  four views  ")
	if request.Prompt != "four views" || request.Size != "1536x1024" || request.Transparent {
		t.Fatalf("request=%#v", request)
	}
}
