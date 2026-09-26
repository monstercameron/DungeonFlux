package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// Start stores the immutable run seed and run metadata.
func (s *Store) Start(ctx context.Context, run domain.Run) error {
	if s == nil || s.writer == nil {
		return fmt.Errorf("start run: closed store")
	}
	seed := append([]byte(nil), run.Seed...)
	return s.writer.submit(ctx, func(ctx context.Context, conn *sql.Conn) error {
		_, err := conn.ExecContext(ctx, `INSERT INTO runs(id, room, mode, seed) VALUES (?, ?, ?, ?)`, string(run.ID), string(run.Room), run.Mode, seed)
		if err != nil {
			return fmt.Errorf("insert run %q: %w", run.ID, err)
		}
		return nil
	})
}
