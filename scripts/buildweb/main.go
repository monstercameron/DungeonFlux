// Command buildweb compresses the browser WASM bundle with deterministic gzip.
package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("buildweb", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	input := flags.String("input", "", "uncompressed input path")
	output := flags.String("output", "", "gzip output path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *input == "" || *output == "" {
		return fmt.Errorf("-input and -output are required")
	}
	return compress(*input, *output)
}

func compress(input, output string) error {
	source, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer func() { _ = source.Close() }()
	target, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	completed := false
	defer func() {
		if !completed {
			_ = target.Close()
			_ = os.Remove(output)
		}
	}()
	writer := gzip.NewWriter(target)
	writer.Header.ModTime = writer.Header.ModTime.UTC()
	if _, err := io.Copy(writer, source); err != nil {
		_ = writer.Close()
		return fmt.Errorf("compress input: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish gzip: %w", err)
	}
	if err := target.Close(); err != nil {
		return fmt.Errorf("close output: %w", err)
	}
	completed = true
	return nil
}
