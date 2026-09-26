package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const uiTrimColour = uint32(0x0f1117ff)

// UIExpectedNames is the build-time UI art set required by the demo.
var UIExpectedNames = []string{
	"banner_callout", "button_disabled", "button_primary", "button_secondary", "check_backdrop",
	"class_bard", "class_cleric", "class_paladin", "class_rogue", "class_wizard", "cliffhanger",
	"d20", "d20_fail", "d20_success", "divider", "end_bg", "icon_attack", "icon_end_turn",
	"icon_leave", "icon_move", "icon_persuade", "icon_ready", "icon_step_away", "icon_talk",
	"lobby_bg", "logo_emblem", "logo_wordmark", "panel_frame", "panel_texture", "phone_bg",
	"qr_frame", "species_dwarf", "species_elf", "species_halfling", "species_human", "species_orc",
	"species_tiefling", "status_bloodied", "status_down", "status_spotlight", "title_bg", "title_bg_wide",
}

func init() {
	if len(os.Args) < 2 || os.Args[1] != "ui" {
		return
	}
	if err := runUI(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func runUI(args []string) error {
	flags := flag.NewFlagSet("ui", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", "artifacts/runtime/buildtime", "build-time output directory")
	check := flags.String("check", "", "comma-separated names or a file containing expected names")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*check) != "" {
		expected, err := readUIExpected(*check)
		if err != nil {
			return err
		}
		missing, err := CheckUI(*root, expected)
		if err != nil {
			return err
		}
		for _, name := range missing {
			fmt.Println(name)
		}
		if len(missing) > 0 {
			return fmt.Errorf("buildtime: %d UI assets missing", len(missing))
		}
		return nil
	}
	return ConvertUI(context.Background(), *root)
}

func readUIExpected(value string) ([]string, error) {
	if data, err := os.ReadFile(value); err == nil {
		lines := strings.Split(string(data), "\n")
		return lines, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read expected UI list: %w", err)
	}
	return strings.Split(value, ","), nil
}

// ConvertUI converts every UI PNG to a quality-82 WebP and registers it.
func ConvertUI(ctx context.Context, root string) error {
	if strings.TrimSpace(root) == "" {
		return errors.New("buildtime: empty UI root")
	}
	uiRoot := filepath.Join(root, "ui")
	entries, err := os.ReadDir(uiRoot)
	if err != nil {
		return fmt.Errorf("read UI directory: %w", err)
	}
	writer, err := NewManifestWriter(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".png") {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		pngPath := filepath.Join(uiRoot, entry.Name())
		webpPath := filepath.Join(uiRoot, strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))+".webp")
		if err := convertUIFile(ctx, pngPath, webpPath); err != nil {
			return fmt.Errorf("convert %q: %w", entry.Name(), err)
		}
		if err := registerUIFile(writer, webpPath); err != nil {
			return fmt.Errorf("register %q: %w", entry.Name(), err)
		}
	}
	if _, err := writer.Write(); err != nil {
		return fmt.Errorf("write UI manifest: %w", err)
	}
	return nil
}

func convertUIFile(ctx context.Context, source, destination string) error {
	input := source
	var temporary string
	defer func() {
		if temporary != "" {
			_ = os.Remove(temporary)
		}
	}()
	if shouldTrimUI(filepath.Base(source)) {
		trimmed, err := trimUIPNG(source)
		if err != nil {
			return err
		}
		input = trimmed
		if trimmed != source {
			temporary = trimmed
		}
	}
	width, height, err := imageSize(input)
	if err != nil {
		return err
	}
	width, height = cappedSize(width, height, 1920)
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	args := []string{"-y", "-v", "error", "-i", input, "-vf", "scale=" + strconv.Itoa(width) + ":" + strconv.Itoa(height), "-c:v", "libwebp", "-quality", "82", destination}
	if output, err := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func registerUIFile(writer *ManifestWriter, source string) error {
	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	take, err := writer.AddFile(base, "IMAGE", source, 1)
	if err != nil {
		return err
	}
	writer.mu.Lock()
	delete(writer.manifest.Assets, base)
	writer.manifest.Assets["ui/"+base] = Asset{Kind: "IMAGE", Selected: 1, Takes: []Take{take}}
	writer.mu.Unlock()
	return nil
}

// CheckUI returns the expected logical names absent from the UI directory.
func CheckUI(root string, expected []string) ([]string, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("buildtime: empty UI root")
	}
	missing := make([]string, 0)
	for _, name := range expected {
		name = strings.TrimPrefix(strings.TrimSpace(name), "ui/")
		if name == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "ui", name+".webp")); errors.Is(err, os.ErrNotExist) {
			missing = append(missing, "ui/"+name)
		} else if err != nil {
			return nil, fmt.Errorf("stat UI asset %q: %w", name, err)
		}
	}
	sort.Strings(missing)
	return missing, nil
}

func imageSize(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("open image: %w", err)
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("decode image: %w", err)
	}
	return config.Width, config.Height, nil
}

func cappedSize(width, height, maximum int) (int, int) {
	if width <= maximum && height <= maximum {
		return width, height
	}
	if width >= height {
		return maximum, maximum * height / width
	}
	return maximum * width / height, maximum
}

func shouldTrimUI(name string) bool {
	for _, prefix := range []string{"icon_", "class_", "status_", "logo_", "button_", "banner_"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return name == "qr_frame.png" || name == "divider.png"
}

// trimUIPNG removes only exact #0f1117 margins and returns a temporary PNG.
func trimUIPNG(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open trim source: %w", err)
	}
	defer file.Close()
	decoded, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("decode trim source: %w", err)
	}
	bounds := decoded.Bounds()
	left, top, right, bottom := bounds.Min.X, bounds.Min.Y, bounds.Max.X-1, bounds.Max.Y-1
	for left < right && columnColour(decoded, left, top, bottom) == uiTrimColour {
		left++
	}
	for right > left && columnColour(decoded, right, top, bottom) == uiTrimColour {
		right--
	}
	for top < bottom && rowColour(decoded, top, left, right) == uiTrimColour {
		top++
	}
	for bottom > top && rowColour(decoded, bottom, left, right) == uiTrimColour {
		bottom--
	}
	if left == bounds.Min.X && top == bounds.Min.Y && right == bounds.Max.X-1 && bottom == bounds.Max.Y-1 {
		return path, nil
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".ui-trim-*.png")
	if err != nil {
		return "", fmt.Errorf("create trim output: %w", err)
	}
	name := temporary.Name()
	defer temporary.Close()
	if err := encodePNG(temporary, decoded, image.Rect(left, top, right+1, bottom+1)); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}

func encodePNG(file *os.File, source image.Image, crop image.Rectangle) error {
	canvas := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	draw.Draw(canvas, canvas.Bounds(), source, crop.Min, draw.Src)
	if err := png.Encode(file, canvas); err != nil {
		return fmt.Errorf("encode trimmed PNG: %w", err)
	}
	return nil
}

func pixelColour(img image.Image, x, y int) uint32 {
	r, g, b, a := img.At(x, y).RGBA()
	return uint32(r>>8)<<24 | uint32(g>>8)<<16 | uint32(b>>8)<<8 | uint32(a>>8)
}
func columnColour(img image.Image, x, top, bottom int) uint32 {
	for y := top; y <= bottom; y++ {
		if pixelColour(img, x, y) != uiTrimColour {
			return 0
		}
	}
	return uiTrimColour
}
func rowColour(img image.Image, y, left, right int) uint32 {
	for x := left; x <= right; x++ {
		if pixelColour(img, x, y) != uiTrimColour {
			return 0
		}
	}
	return uiTrimColour
}
