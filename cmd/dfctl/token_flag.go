package main

import "flag"

// bindTokenFlag preserves the environment value without registering it as a
// printable default. Flag usage must never expose an authentication token.
func bindTokenFlag(flags *flag.FlagSet, token *string) {
	flags.Func("token", "debug token (defaults to DF_DEBUG_TOKEN)", func(value string) error {
		*token = value
		return nil
	})
}
