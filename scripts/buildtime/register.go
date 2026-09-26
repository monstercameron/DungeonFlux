package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	return writer.Write()
}
