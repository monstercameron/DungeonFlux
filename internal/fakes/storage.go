package fakes

import (
	"context"
	"iter"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// InboxCall records an envelope posted to FakeInbox.
type InboxCall struct {
	Context  context.Context
	Envelope domain.Envelope
}

// FakeInbox records posts and returns PostResult (true by default).
type FakeInbox struct {
	PostResult bool
	Calls      []InboxCall
	mu         sync.Mutex
}

// Post records env and returns whether the inbox accepted it.
func (f *FakeInbox) Post(ctx context.Context, env domain.Envelope) bool {
	f.mu.Lock()
	f.Calls = append(f.Calls, InboxCall{ctx, env})
	ok := f.PostResult
	if len(f.Calls) == 1 && !f.PostResult {
		ok = false
	}
	f.mu.Unlock()
	select {
	case <-ctx.Done():
		return false
	default:
		return ok
	}
}

// AudioFrameCall records one audio frame.
type AudioFrameCall struct{ Frame domain.AudioFrame }

// AudioCancelCall records one audio cancellation.
type AudioCancelCall struct{ UtteranceID domain.UtteranceID }

// FakeAudioOut records frames and cancellation requests.
type FakeAudioOut struct {
	Frames  []AudioFrameCall
	Cancels []AudioCancelCall
	mu      sync.Mutex
}

// Frame records f.
func (f *FakeAudioOut) Frame(frame domain.AudioFrame) {
	f.mu.Lock()
	f.Frames = append(f.Frames, AudioFrameCall{frame})
	f.mu.Unlock()
}

// Cancel records id.
func (f *FakeAudioOut) Cancel(id domain.UtteranceID) {
	f.mu.Lock()
	f.Cancels = append(f.Cancels, AudioCancelCall{id})
	f.mu.Unlock()
}

// AssetWriteCall records one asset write.
type AssetWriteCall struct {
	Kind vocab.AssetKind
	MIME string
	Data []byte
	Meta ports.AssetMeta
}

// AssetWriteResult scripts one asset write.
type AssetWriteResult struct {
	Asset domain.Asset
	Err   error
}

// FakeAssetWriter is a scripted asset writer.
type FakeAssetWriter struct {
	Script []AssetWriteResult
	Calls  []AssetWriteCall
	mu     sync.Mutex
}

// Write records the bytes and returns the next scripted result.
func (f *FakeAssetWriter) Write(_ context.Context, kind vocab.AssetKind, mime string, data []byte, meta ports.AssetMeta) (domain.Asset, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, AssetWriteCall{kind, mime, append([]byte(nil), data...), meta})
	r := AssetWriteResult{}
	if len(f.Script) > 0 {
		r, f.Script = f.Script[0], f.Script[1:]
	}
	return r.Asset, r.Err
}

// FakeEventLog is an in-memory event log with scriptable append errors.
type FakeEventLog struct {
	Records      []domain.LogRecord
	AppendErrors []error
	AppendCalls  [][]domain.LogRecord
	mu           sync.Mutex
}

// Append stores records unless the next scripted error is non-nil.
func (f *FakeEventLog) Append(_ context.Context, records []domain.LogRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	x := append([]domain.LogRecord(nil), records...)
	f.AppendCalls = append(f.AppendCalls, x)
	var err error
	if len(f.AppendErrors) > 0 {
		err, f.AppendErrors = f.AppendErrors[0], f.AppendErrors[1:]
	}
	if err == nil {
		f.Records = append(f.Records, x...)
	}
	return err
}

// Read returns records for run until the consumer stops.
func (f *FakeEventLog) Read(ctx context.Context, run domain.RunID) iter.Seq2[domain.LogRecord, error] {
	f.mu.Lock()
	rows := append([]domain.LogRecord(nil), f.Records...)
	f.mu.Unlock()
	return func(yield func(domain.LogRecord, error) bool) {
		for _, r := range rows {
			if r.Run != run {
				continue
			}
			select {
			case <-ctx.Done():
				yield(domain.LogRecord{}, ctx.Err())
				return
			default:
			}
			if !yield(r, nil) {
				return
			}
		}
	}
}

// FakeRuns records started runs and returns scripted errors.
type FakeRuns struct {
	Runs   []domain.Run
	Errors []error
	mu     sync.Mutex
}

// Start records r and returns the next scripted error.
func (f *FakeRuns) Start(_ context.Context, r domain.Run) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Runs = append(f.Runs, r)
	if len(f.Errors) == 0 {
		return nil
	}
	e := f.Errors[0]
	f.Errors = f.Errors[1:]
	return e
}

// FakeAssets is an in-memory asset store keyed by SHA256.
type FakeAssets struct {
	Values map[string]domain.Asset
	Errors []error
	mu     sync.Mutex
}

// Put stores an asset, keyed by its SHA256.
func (f *FakeAssets) Put(_ context.Context, a domain.Asset) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e := popError(&f.Errors); e != nil {
		return e
	}
	if f.Values == nil {
		f.Values = make(map[string]domain.Asset)
	}
	f.Values[a.SHA256] = a
	return nil
}

// Get returns an asset and whether it was present.
func (f *FakeAssets) Get(_ context.Context, sha string) (domain.Asset, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e := popError(&f.Errors); e != nil {
		return domain.Asset{}, false, e
	}
	a, ok := f.Values[sha]
	return a, ok, nil
}

// FakeCache is an in-memory adapter/hash byte cache.
type FakeCache struct {
	Values map[string][]byte
	Errors []error
	mu     sync.Mutex
}

// Get returns cached bytes and whether they were present.
func (f *FakeCache) Get(_ context.Context, adapter, hash string) ([]byte, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e := popError(&f.Errors); e != nil {
		return nil, false, e
	}
	v, ok := f.Values[adapter+"\x00"+hash]
	return append([]byte(nil), v...), ok, nil
}

// Put stores bytes under adapter and hash.
func (f *FakeCache) Put(_ context.Context, adapter, hash string, v []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e := popError(&f.Errors); e != nil {
		return e
	}
	if f.Values == nil {
		f.Values = make(map[string][]byte)
	}
	f.Values[adapter+"\x00"+hash] = append([]byte(nil), v...)
	return nil
}

// FakeRecordings is an in-memory recording store.
type FakeRecordings struct {
	Values map[ports.RecKey]domain.Recording
	Errors []error
	mu     sync.Mutex
}

// Get returns a recording and whether it was present.
func (f *FakeRecordings) Get(_ context.Context, key ports.RecKey) (domain.Recording, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e := popError(&f.Errors); e != nil {
		return domain.Recording{}, false, e
	}
	r, ok := f.Values[key]
	return r, ok, nil
}

// Put stores a recording.
func (f *FakeRecordings) Put(_ context.Context, key ports.RecKey, r domain.Recording) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e := popError(&f.Errors); e != nil {
		return e
	}
	if f.Values == nil {
		f.Values = make(map[ports.RecKey]domain.Recording)
	}
	f.Values[key] = r
	return nil
}

func popError(errors *[]error) error {
	if len(*errors) == 0 {
		return nil
	}
	e := (*errors)[0]
	*errors = (*errors)[1:]
	return e
}
