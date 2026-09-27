package main

import (
	"io"
	"strings"
	"testing"
)

func TestTokenFlag_UsageNeverExposesCredentials(t *testing.T) {
	t.Setenv("DF_DEBUG_TOKEN", "environment-test-secret")
	for _, parser := range tokenParsers() {
		t.Run(parser.name, func(t *testing.T) {
			for _, args := range [][]string{{"--help"}, {"--unknown"}, {"--token", "explicit-test-secret", "--help"}} {
				var output strings.Builder
				if _, err := parser.parse(args, &output); err == nil {
					t.Fatal("help or invalid option unexpectedly succeeded")
				}
				for _, secret := range []string{"environment-test-secret", "explicit-test-secret"} {
					if strings.Contains(output.String(), secret) {
						t.Fatal("usage exposed a credential")
					}
				}
				if !strings.Contains(output.String(), "-token") {
					t.Fatal("usage no longer documents the token option")
				}
			}
		})
	}
}

func TestTokenFlag_EnvironmentAndExplicitPrecedence(t *testing.T) {
	t.Setenv("DF_DEBUG_TOKEN", "environment-test-secret")
	for _, parser := range tokenParsers() {
		t.Run(parser.name, func(t *testing.T) {
			for _, tc := range []struct {
				args []string
				want string
			}{{nil, "environment-test-secret"}, {[]string{"--token", "explicit-test-secret"}, "explicit-test-secret"}} {
				opts, err := parser.parse(tc.args, io.Discard)
				if err != nil || opts.token != tc.want {
					t.Fatal("token precedence changed or parsing failed")
				}
			}
		})
	}
}

type tokenParser struct {
	name  string
	parse func([]string, io.Writer) (options, error)
}

func tokenParsers() []tokenParser {
	return []tokenParser{
		{"base", func(args []string, out io.Writer) (options, error) {
			opts, _, err := parseOptions(append(args, "state"), out)
			return opts, err
		}},
		{"read", func(args []string, out io.Writer) (options, error) {
			opts, _, err := parseReadOptions(append(args, "state"), "state", out)
			return opts, err
		}},
		{"control", func(args []string, out io.Writer) (options, error) {
			opts, _, _, err := parseControlOptions(append(args, "pause"), "pause", out)
			return opts, err
		}},
		{"write", func(args []string, out io.Writer) (options, error) {
			opts, _, err := parseWriteOptions(append(args, "reset"), out)
			return opts, err
		}},
	}
}
