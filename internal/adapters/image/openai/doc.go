// Package openai adapts the OpenAI Images API to the DungeonFlux image port.
// It owns request encoding, response decoding, and vendor error mapping; the
// composition root supplies the API key and shared HTTP client configuration.
package openai
