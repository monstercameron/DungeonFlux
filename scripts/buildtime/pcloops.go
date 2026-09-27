package main

import (
	"context"
	"errors"
	"fmt"
)

// PCBillboardSpec identifies the four status loops for one PC template.
type PCBillboardSpec struct {
	Template   string
	FirstFrame string
}

// PCBillboardSpecs returns the two demo templates and their combat poses.
func PCBillboardSpecs() []LoopSpec {
	poses := []struct{ name, prompt string }{
		{"idle", "full-body fantasy hero breathing and shifting weight, static camera"},
		{"attack", "full-body fantasy hero performs one weapon swing toward screen right, static camera"},
		{"hit", "full-body fantasy hero recoils from an impact, static camera"},
		{"down", "full-body fantasy hero holds a defeated standing pose, static camera"},
	}
	var specs []LoopSpec
	for _, template := range []string{"pc_template_1", "pc_template_2"} {
		for _, pose := range poses {
			specs = append(specs, LoopSpec{LogicalName: template + "_" + pose.name, Prompt: pose.prompt, Take: 1, DurationMS: 4000, ContactMS: 1200})
		}
	}
	return specs
}

// RunPCBillboardJob renders the PC pose loops with a 1.2-second contact.
func RunPCBillboardJob(ctx context.Context, writer *ManifestWriter, options LoopOptions) error {
	if writer == nil {
		return errors.New("buildtime: nil manifest writer")
	}
	if len(options.Specs) == 0 {
		options.Specs = PCBillboardSpecs()
	}
	for i := range options.Specs {
		if options.Specs[i].ContactMS == 0 {
			options.Specs[i].ContactMS = 1200
		}
		if options.Specs[i].DurationMS == 0 {
			options.Specs[i].DurationMS = 4000
		}
	}
	if options.DryRun {
		fmt.Printf("dry-run PC billboard loops: %d\n", len(options.Specs))
	}
	return RunLoopJob(ctx, writer, options, "VIDEO_PC_LOOP")
}
