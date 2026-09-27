package ports

import "context"

type callMetaKey struct{}

// WithCallMeta binds the room-assigned rehearsal position to one work context.
// The value is copied, so executors cannot mutate the room's sequence state.
func WithCallMeta(ctx context.Context, meta CallMeta) context.Context {
	return context.WithValue(ctx, callMetaKey{}, meta)
}

// ResolveCallMeta applies the room's position while retaining request-specific
// language, role, utterance and replay policy supplied by adapter decorators.
func ResolveCallMeta(ctx context.Context, requested CallMeta) CallMeta {
	bound, ok := ctx.Value(callMetaKey{}).(CallMeta)
	if !ok {
		return requested
	}
	requested.Run, requested.Phase, requested.Index = bound.Run, bound.Phase, bound.Index
	if bound.Seat != 0 {
		requested.Seat = bound.Seat
	}
	if requested.Role == "" {
		requested.Role = bound.Role
	}
	if requested.UtteranceID == "" {
		requested.UtteranceID = bound.UtteranceID
	}
	requested.Speculative = requested.Speculative || bound.Speculative
	return requested
}
