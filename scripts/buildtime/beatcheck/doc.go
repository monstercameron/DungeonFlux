// Command beatcheck measures tempo and the first onset in decoded audio.
//
// The command intentionally has no audio-library dependencies. ffmpeg decodes
// the input to mono PCM, while the analyzer uses a short-time energy envelope
// to find regular click-track onsets.
package main
