package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type scannedAsset struct {
	file    string
	logical string
	kind    string
}

var stillRegistry = []scannedAsset{
	{file: "battlefield_flat.png", logical: "battlefield_tavern_flat", kind: "IMAGE"},
	{file: "bell_tower.png", logical: "cliff_generic_tower", kind: "IMAGE"},
	{file: "mother_vell_source.png", logical: "mother_vell", kind: "IMAGE"},
	{file: "stranger_source.png", logical: "stranger", kind: "IMAGE"},
	{file: "tavern_doorway.png", logical: "arrival_door", kind: "IMAGE"},
	{file: "tavern_interior.png", logical: "establishing_tavern", kind: "IMAGE"},
}

// RegisterScannedStills adds known generated stills to root's manifest.
func RegisterScannedStills(root string) (string, error) {
	if root == "" {
		return "", errors.New("buildtime: empty scan directory")
	}
	writer, err := NewManifestWriter(root)
	if err != nil {
		return "", err
	}
	for _, asset := range stillRegistry {
		source := filepath.Join(root, asset.file)
		if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return "", fmt.Errorf("stat still %q: %w", asset.file, err)
		}
		if _, err := writer.AddFile(asset.logical, asset.kind, source, 1); err != nil {
			return "", fmt.Errorf("register still %q: %w", asset.file, err)
		}
	}
	if err := registerScannedAudio(root, writer); err != nil {
		return "", err
	}
	return writer.Write()
}

type scannedAudio struct {
	directory string
	logical   string
	kind      string
}

var audioRegistry = []scannedAudio{
	{directory: "sfx", logical: "sfx_join_tv", kind: "SFX"},
	{directory: "sfx", logical: "sfx_phone_confirm", kind: "SFX"},
	{directory: "sfx", logical: "sfx_ready", kind: "SFX"},
	{directory: "sfx", logical: "sfx_host_start", kind: "SFX"},
	{directory: "sfx", logical: "sfx_phone_tick", kind: "SFX"},
	{directory: "sfx", logical: "sfx_phone_dice", kind: "SFX"},
	{directory: "sfx", logical: "sfx_roll_reveal", kind: "SFX"},
	{directory: "sfx", logical: "sfx_hero_lock", kind: "SFX"},
	{directory: "sfx", logical: "sfx_dice_roll", kind: "SFX"},
	{directory: "sfx", logical: "sfx_check_success", kind: "SFX"},
	{directory: "sfx", logical: "sfx_check_failure", kind: "SFX"},
	{directory: "sfx", logical: "sfx_door_burst", kind: "SFX"},
	{directory: "sfx", logical: "sfx_tavern_ambience", kind: "SFX"},
	{directory: "sfx", logical: "sfx_stranger_sting", kind: "SFX"},
	{directory: "sfx", logical: "sfx_cliffhanger_hit", kind: "SFX"},
	{directory: "sfx", logical: "sfx_sword_slash", kind: "SFX"},
	{directory: "sfx", logical: "sfx_blade_hit", kind: "SFX"},
	{directory: "sfx", logical: "sfx_mace_thud", kind: "SFX"},
	{directory: "sfx", logical: "sfx_dagger_stab", kind: "SFX"},
	{directory: "sfx", logical: "sfx_miss_whoosh", kind: "SFX"},
	{directory: "sfx", logical: "sfx_slam_impact", kind: "SFX"},
	{directory: "sfx", logical: "sfx_thrall_groan", kind: "SFX"},
	{directory: "sfx", logical: "sfx_splash_collapse", kind: "SFX"},
	{directory: "sfx", logical: "sfx_wet_footsteps", kind: "SFX"},
	{directory: "music", logical: "THEME_MAIN", kind: "MUSIC"},
	{directory: "music", logical: "CREATION_BED_LOOP", kind: "MUSIC"},
	{directory: "music", logical: "OPENING_SWELL", kind: "MUSIC"},
	{directory: "music", logical: "TAVERN_WARM_LOOP", kind: "MUSIC"},
	{directory: "music", logical: "STING_STRANGER", kind: "MUSIC"},
	{directory: "music", logical: "STING_COMBAT_START", kind: "MUSIC"},
	{directory: "music", logical: "COMBAT_SKIRMISH_LOOP", kind: "MUSIC"},
	{directory: "music", logical: "STING_VICTORY", kind: "MUSIC"},
	{directory: "music", logical: "STING_BELL_TOLL", kind: "MUSIC"},
	{directory: "music", logical: "CLIFF_TENSION_BED", kind: "MUSIC"},
	{directory: "music", logical: "STING_CLIFF_HIT", kind: "MUSIC"},
	{directory: "music", logical: "END_CARD_THEME", kind: "MUSIC"},
	{directory: "audio", logical: "canned_opening", kind: "AUDIO"},
	{directory: "audio", logical: "canned_npc_reply", kind: "AUDIO"},
	{directory: "audio", logical: "canned_npc_reveal", kind: "AUDIO"},
	{directory: "audio", logical: "canned_npc_refuse", kind: "AUDIO"},
	{directory: "audio", logical: "canned_stranger_found", kind: "AUDIO"},
	{directory: "audio", logical: "canned_stranger_relocated", kind: "AUDIO"},
	{directory: "audio", logical: "canned_cliffhanger_vell", kind: "AUDIO"},
	{directory: "audio", logical: "canned_cliffhanger_stranger", kind: "AUDIO"},
	{directory: "audio", logical: "canned_combat_slain_seat1", kind: "AUDIO"},
	{directory: "audio", logical: "canned_combat_slain_seat2", kind: "AUDIO"},
	{directory: "audio", logical: "canned_combat_fled", kind: "AUDIO"},
	{directory: "audio", logical: "nudge_exploration", kind: "AUDIO"},
	{directory: "audio", logical: "nudge_conversation", kind: "AUDIO"},
}

func registerScannedAudio(root string, writer *ManifestWriter) error {
	for _, asset := range audioRegistry {
		for _, pattern := range []string{asset.logical + "_take*", asset.logical + "-take-*"} {
			files, err := filepath.Glob(filepath.Join(root, asset.directory, pattern))
			if err != nil {
				return fmt.Errorf("buildtime: scan audio %q: %w", asset.logical, err)
			}
			for _, source := range files {
				take, ok := audioTakeNumber(source, asset.logical)
				if !ok {
					continue
				}
				if _, err := writer.AddFile(asset.logical, asset.kind, source, take); err != nil {
					return fmt.Errorf("register audio %q: %w", source, err)
				}
			}
		}
	}
	return nil
}

func audioTakeNumber(path, logical string) (int, bool) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	for _, prefix := range []string{logical + "_take", logical + "-take-"} {
		if strings.HasPrefix(name, prefix) {
			take, err := strconv.Atoi(strings.TrimPrefix(name, prefix))
			return take, err == nil && take > 0
		}
	}
	return 0, false
}
