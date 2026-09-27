package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/media"
	"github.com/monstercameron/DungeonFlux/internal/wire"
)

// thrallLook and thrallWeapon describe the drowned thrall to the video
// model (name-free, §0.17).
const (
	thrallLook   = "drowned thrall, a waterlogged undead sailor bound to a tower bell,"
	thrallWeapon = "rusted length of bell chain"
	thrallRef    = "thrall_still"
)

// thrallRefs are the manifest entries that can serve as the thrall's
// identity image, best first: the transparent cut-out, then the opaque still.
var thrallRefs = []string{"thrall_cutout", thrallRef}

// thrallRefPrompt is the one-off Images API prompt for the thrall's identity
// still when the manifest has neither a cut-out nor a still. The Images API
// may return it opaque, so it is registered as thrall_still.
const thrallRefPrompt = "Full-body character reference of a drowned thrall: a waterlogged undead sailor bound to an old tower bell, grey-green swollen skin, torn dark coat hung with river weed, a rusted length of bell chain wrapped around one forearm, hollow pale eyes, standing upright facing the viewer with arms at its sides, the whole body from head to feet in frame, painterly dark-fantasy illustration, visible brushstrokes, muted teal and amber palette, no text"

// billboardFlags are the options of the billboards subcommand.
type billboardFlags struct {
	root, dataDir, level, refs, look, weapon, actions, register string
	thrall                                                      bool
	maxUSD                                                      float64
}

// billboardLine is one JSON result line printed per loop.
type billboardLine struct {
	Action    string  `json:"action"`
	Key       string  `json:"cache_key"`
	SHA256    string  `json:"sha256"`
	Path      string  `json:"path"`
	Cached    bool    `json:"cached"`
	LatencyMS int64   `json:"latency_ms"`
	Calls     int64   `json:"fal_calls_total"`
	SpentUSD  float64 `json:"spent_usd_total"`
	Logical   string  `json:"registered_as,omitempty"`
}

func parseBillboardFlags(args []string) (billboardFlags, error) {
	var options billboardFlags
	flags := flag.NewFlagSet("billboards", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&options.root, "root", "artifacts/runtime/buildtime", "build-time output directory")
	flags.StringVar(&options.dataDir, "data-dir", "artifacts/runtime/buildtime/billboard-cache", "cache database and asset store (a server data dir works too)")
	flags.StringVar(&options.level, "level", "level_still_64bb46d5_tactical", "manifest name of the level still")
	flags.StringVar(&options.refs, "refs", "", "comma-separated identity images, front first (files or manifest:<name>)")
	flags.StringVar(&options.look, "look", "", "short name-free look descriptor")
	flags.StringVar(&options.weapon, "weapon", "", "weapon for the attack loop")
	flags.StringVar(&options.actions, "actions", "idle,attack,hit,fall", "comma-separated loop actions")
	flags.StringVar(&options.register, "register", "", "register each loop in the manifest as <prefix><action>")
	flags.BoolVar(&options.thrall, "thrall", false, "the drowned thrall: refs, look, weapon and thrall_loop_ registration")
	flags.Float64Var(&options.maxUSD, "max-usd", 2, "fal spend cap for this invocation")
	if err := flags.Parse(args); err != nil {
		return billboardFlags{}, err
	}
	if options.thrall {
		options.refs = firstNonEmpty(options.refs, "manifest:"+thrallRefs[0]+"|"+thrallRefs[1])
		options.look = firstNonEmpty(options.look, thrallLook)
		options.weapon = firstNonEmpty(options.weapon, thrallWeapon)
		options.register = firstNonEmpty(options.register, "thrall_loop_")
	}
	if options.refs == "" {
		return billboardFlags{}, errors.New("buildtime: billboards needs --refs or --thrall")
	}
	for _, action := range splitList(options.actions) {
		if !media.BillboardActions(action) {
			return billboardFlags{}, fmt.Errorf("buildtime: unknown loop action %q", action)
		}
	}
	return options, nil
}

// runBillboards renders (or finds in the cache) each requested loop and
// optionally registers it in the manifest with its contact frame.
func runBillboards(ctx context.Context, args []string, out io.Writer) error {
	options, err := parseBillboardFlags(args)
	if err != nil {
		return err
	}
	if options.thrall {
		if err := ensureThrallReference(ctx, options.root); err != nil {
			return err
		}
	}
	spec, err := billboardSubjectSpec(options)
	if err != nil {
		return err
	}
	tool, err := wire.OpenBillboardTool(ctx, options.dataDir, envKey("DF_FAL_KEY"), options.maxUSD, nil)
	if err != nil {
		return err
	}
	defer tool.Close()
	actions := splitList(options.actions)
	lines, err := generateLoops(ctx, tool, spec, actions)
	for _, line := range lines {
		if encodeErr := json.NewEncoder(out).Encode(line); encodeErr != nil {
			return encodeErr
		}
	}
	if err != nil {
		return err
	}
	if options.register == "" {
		return nil
	}
	return registerLoops(options.root, options.register, lines)
}

// generateLoops renders every action at once; the tool's fal pool admits
// two at a time. Lines come back in action order.
func generateLoops(ctx context.Context, tool *wire.BillboardTool, spec media.BillboardSpec, actions []string) ([]billboardLine, error) {
	lines := make([]billboardLine, len(actions))
	errs := make([]error, len(actions))
	var group sync.WaitGroup
	for index, action := range actions {
		group.Go(func() {
			one := spec
			one.Action = action
			result, err := tool.Generate(ctx, one)
			if err != nil {
				errs[index] = fmt.Errorf("buildtime: %s loop: %w", action, err)
				return
			}
			lines[index] = billboardLine{Action: action, Key: result.Key, SHA256: result.Asset.SHA256, Path: tool.AssetPath(result.Asset),
				Cached: result.Cached, LatencyMS: result.Latency.Milliseconds()}
		})
	}
	group.Wait()
	done := lines[:0]
	for index, line := range lines {
		if errs[index] == nil {
			line.Calls, line.SpentUSD = tool.VendorCalls(), tool.SpentUSD()
			done = append(done, line)
		}
	}
	return done, errors.Join(errs...)
}

func billboardSubjectSpec(options billboardFlags) (media.BillboardSpec, error) {
	level, err := manifestBytes(options.root, options.level)
	if err != nil {
		return media.BillboardSpec{}, fmt.Errorf("buildtime: level still %q: %w", options.level, err)
	}
	subject := media.BillboardSubject{Look: options.look, Weapon: options.weapon}
	for _, ref := range splitList(options.refs) {
		var data []byte
		if names, ok := strings.CutPrefix(ref, "manifest:"); ok {
			data, err = firstManifestBytes(options.root, strings.Split(names, "|"))
		} else {
			data, err = os.ReadFile(ref)
		}
		if err != nil {
			return media.BillboardSpec{}, fmt.Errorf("buildtime: reference %q: %w", ref, err)
		}
		subject.References = append(subject.References, data)
	}
	return media.BillboardSpec{Subject: subject, LevelStill: level}, nil
}

// registerLoops re-reads the manifest just before writing, so other jobs'
// entries written during the renders are kept.
func registerLoops(root, prefix string, lines []billboardLine) error {
	writer, err := NewManifestWriter(root)
	if err != nil {
		return err
	}
	for index, line := range lines {
		logical := prefix + line.Action
		if _, err := writer.AddFile(logical, "VIDEO_LOOP", line.Path, 1); err != nil {
			return err
		}
		if err := writer.SelectTake(logical, 1); err != nil {
			return err
		}
		metadata := map[string]string{"model": "bytedance/seedance-2.0/fast/reference-to-video", "resolution": media.BillboardResolution,
			"aspect": media.BillboardAspect, "no_audio": "true", "cache_key": line.Key, "prompt_version": media.BillboardPromptVersion, "contact_source": "default-1.2s"}
		if line.Action == media.BillboardIdle {
			metadata["pinned"] = "prompt"
		}
		if err := writer.SetMetadata(logical, int64(media.BillboardSeconds*1000), loopContactMS(line.Action), metadata); err != nil {
			return err
		}
		lines[index].Logical = logical
	}
	_, err = writer.Write()
	return err
}

// loopContactMS is the §0.21.5 default contact frame: no measuring helper
// exists yet, so attack and hit loops take 1.2 s.
func loopContactMS(action string) int64 {
	if action == media.BillboardAttack || action == media.BillboardHit {
		return media.BillboardContactMS
	}
	return 0
}

// runLevelStill registers a clean level still captured from the standalone
// viewer (grid hidden, no tokens, no UI) as level_still_<scene>_<preset>.
func runLevelStill(args []string) error {
	flags := flag.NewFlagSet("level-still", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", "artifacts/runtime/buildtime", "build-time output directory")
	scene := flags.String("scene", "", "scene id, e.g. 64bb46d5")
	preset := flags.String("preset", "tactical", "camera preset")
	file := flags.String("file", "", "captured PNG")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *scene == "" || *file == "" {
		return errors.New("buildtime: level-still needs --scene and --file")
	}
	writer, err := NewManifestWriter(*root)
	if err != nil {
		return err
	}
	logical := "level_still_" + strings.ToLower(*scene) + "_" + strings.ToLower(*preset)
	if _, err := writer.AddFile(logical, "IMAGE_STILL", *file, 1); err != nil {
		return err
	}
	if err := writer.SetMetadata(logical, 0, 0, map[string]string{"scene": *scene, "preset": strings.ToUpper(*preset), "source": "web/splat/js/viewer.html, grid hidden, no tokens, no UI"}); err != nil {
		return err
	}
	_, err = writer.Write()
	return err
}

// ensureThrallReference generates the thrall's identity image through the
// Images API (≈ $0.04) only when the manifest has none.
func ensureThrallReference(ctx context.Context, root string) error {
	if _, err := firstManifestBytes(root, thrallRefs); err == nil {
		return nil
	}
	writer, err := NewManifestWriter(root)
	if err != nil {
		return err
	}
	options := CutoutOptions{APIKey: envKey("DF_OPENAI_API_KEY"), Specs: []CutoutSpec{{LogicalName: thrallRef, Prompt: thrallRefPrompt, Take: 1}}}
	if err := RunCutoutJob(ctx, writer, options); err != nil {
		return err
	}
	asset := writer.manifest.Assets[thrallRef]
	asset.Kind = "IMAGE_STILL"
	writer.manifest.Assets[thrallRef] = asset
	_, err = writer.Write()
	return err
}

// firstManifestBytes reads the first of names present in the manifest.
func firstManifestBytes(root string, names []string) ([]byte, error) {
	for _, name := range names {
		if data, err := manifestBytes(root, name); err == nil {
			return data, nil
		}
	}
	return nil, os.ErrNotExist
}

// manifestBytes reads the selected take of a manifest entry.
func manifestBytes(root, logical string) ([]byte, error) {
	writer, err := NewManifestWriter(root)
	if err != nil {
		return nil, err
	}
	asset, ok := writer.manifest.Assets[logical]
	if !ok {
		return nil, os.ErrNotExist
	}
	for _, take := range asset.Takes {
		if take.Number == asset.Selected {
			return os.ReadFile(filepath.Join(root, filepath.FromSlash(take.Path)))
		}
	}
	return nil, os.ErrNotExist
}

// envKey reads a key from the environment, else from the repo's .env; the
// value is never printed.
func envKey(name string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	file, err := os.Open(".env")
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if value, ok := strings.CutPrefix(strings.TrimSpace(scanner.Text()), name+"="); ok {
			return strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return ""
}

func splitList(value string) []string {
	var out []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// billboardTimeout bounds one invocation (four loops through two fal slots).
const billboardTimeout = 20 * time.Minute

// runBillboardCommand runs the billboards or level-still subcommand.
func runBillboardCommand(name string, args []string) error {
	if name == "level-still" {
		return runLevelStill(args)
	}
	ctx, cancel := context.WithTimeout(context.Background(), billboardTimeout)
	defer cancel()
	return runBillboards(ctx, args, os.Stdout)
}
