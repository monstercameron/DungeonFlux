package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Job is one independently runnable build-time asset job.
type Job struct {
	Name string
	Run  func(context.Context, *ManifestWriter) error
}

// RunJobs executes jobs, then writes the manifest when all jobs succeed.
func RunJobs(ctx context.Context, writer *ManifestWriter, jobs []Job) error {
	if writer == nil {
		return errors.New("buildtime: nil manifest writer")
	}
	ordered := append([]Job(nil), jobs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Name < ordered[j].Name })
	for _, job := range ordered {
		if strings.TrimSpace(job.Name) == "" || job.Run == nil {
			return errors.New("buildtime: every job needs a name and function")
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := job.Run(ctx, writer); err != nil {
			return fmt.Errorf("job %q: %w", job.Name, err)
		}
	}
	_, err := writer.Write()
	return err
}

func main() {
	root := flag.String("root", "artifacts/runtime/buildtime", "build-time output directory")
	flag.Parse()
	writer, err := NewManifestWriter(*root)
	if err == nil {
		err = RunJobs(context.Background(), writer, nil)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
