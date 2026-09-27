// Package audio receives streamed PCM for the DM screen and schedules it for
// playback through the browser Web Audio API.
//
// The scheduler is platform-neutral and deliberately contains no clock or
// browser state. The js/wasm player is the only part that touches syscall/js.
package audio
