package main

import (
	"context"
	"testing"
)

func TestPCBillboardSpecs_IncludeAllTemplatesAndPoses(t *testing.T) {
	specs := PCBillboardSpecs()
	if len(specs) != 10 {
		t.Fatalf("got %d specs", len(specs))
	}
	seen := map[string]bool{}
	for _, spec := range specs {
		seen[spec.LogicalName] = true
		if spec.DurationMS != 4000 || spec.ContactMS != 1200 {
			t.Fatalf("unexpected timing: %#v", spec)
		}
	}
	for _, name := range []string{"pc_template_1_idle", "pc_template_1_walk", "pc_template_1_attack", "pc_template_1_hit", "pc_template_1_down", "pc_template_2_idle", "pc_template_2_walk", "pc_template_2_attack", "pc_template_2_hit", "pc_template_2_down"} {
		if !seen[name] {
			t.Fatalf("missing %s", name)
		}
	}
}

func TestRunPCBillboardJob_DryRunDoesNotWrite(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := RunPCBillboardJob(context.Background(), writer, LoopOptions{DryRun: true}); err != nil {
		t.Fatal(err)
	}
	if len(writer.manifest.Assets) != 0 {
		t.Fatal("dry-run wrote assets")
	}
}
