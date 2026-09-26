// Package anthropic adapts Claude Haiku's streaming Messages API to ports.LLM.
// It owns SDK request construction, text-delta extraction, and vendor error
// mapping; composition supplies credentials and the HTTP client configuration.
package anthropic
