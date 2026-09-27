package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// AssetStore implements the assets storage port on a shared Store.
type AssetStore struct{ store *Store }

// NewAssets returns an asset storage view over store.
func NewAssets(store *Store) *AssetStore { return &AssetStore{store: store} }

// Put stores asset metadata keyed by its immutable SHA-256.
func (s *AssetStore) Put(ctx context.Context, asset domain.Asset) error {
	if s == nil || s.store == nil || s.store.writer == nil {
		return fmt.Errorf("put asset: closed store")
	}
	meta, err := json.Marshal(asset.Meta)
	if err != nil {
		return fmt.Errorf("encode asset metadata: %w", err)
	}
	return s.store.writer.submit(ctx, func(ctx context.Context, conn *sql.Conn) error {
		_, err := conn.ExecContext(ctx, `INSERT INTO assets(sha256, id, kind, mime, duration_ms, meta_json) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(sha256) DO UPDATE SET id=excluded.id, kind=excluded.kind, mime=excluded.mime, duration_ms=excluded.duration_ms, meta_json=excluded.meta_json`, asset.SHA256, string(asset.ID), asset.Kind, asset.MIME, asset.DurationMS, meta)
		if err != nil {
			return fmt.Errorf("put asset %q: %w", asset.SHA256, err)
		}
		return nil
	})
}

// Get returns asset metadata by SHA-256.
func (s *AssetStore) Get(ctx context.Context, sha string) (domain.Asset, bool, error) {
	if s == nil || s.store == nil || s.store.read == nil {
		return domain.Asset{}, false, fmt.Errorf("get asset: closed store")
	}
	var asset domain.Asset
	var meta []byte
	err := s.store.read.QueryRowContext(ctx, `SELECT id, sha256, kind, mime, duration_ms, meta_json FROM assets WHERE sha256 = ?`, sha).Scan(&asset.ID, &asset.SHA256, &asset.Kind, &asset.MIME, &asset.DurationMS, &meta)
	if err == sql.ErrNoRows {
		return domain.Asset{}, false, nil
	}
	if err != nil {
		return domain.Asset{}, false, fmt.Errorf("get asset %q: %w", sha, err)
	}
	if len(meta) != 0 {
		if err := json.Unmarshal(meta, &asset.Meta); err != nil {
			return domain.Asset{}, false, fmt.Errorf("decode asset %q metadata: %w", sha, err)
		}
	}
	return asset, true, nil
}
