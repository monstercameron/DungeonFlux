// Command spz converts legacy gzip SPZ files to binary little-endian PLY.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/monstercameron/DungeonFlux/scripts/buildtime/spz"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("spz", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	input := flags.String("input", "", "input .spz file")
	output := flags.String("output", "", "full output .ply file")
	lite := flags.String("lite-output", "", "optional 100k opacity-weighted output .ply file")
	limit := flags.Int("max-points", 0, "maximum points for -output; zero keeps all")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *input == "" || *output == "" {
		return fmt.Errorf("-input and -output are required")
	}
	if err := spz.ConvertFile(*input, *output, *limit); err != nil {
		return err
	}
	if *lite != "" {
		if err := spz.ConvertFile(*input, *lite, 100000); err != nil {
			return err
		}
	}
	return nil
}
