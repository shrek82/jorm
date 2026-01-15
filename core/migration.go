package core

import (
	"context"
	"fmt"
)

// Migration represents a single database migration.
type Migration struct {
	Version     int
	Description string
	Up          func(db *DB) error
	Down        func(db *DB) error
}

// Migrator manages database migrations and history.
type Migrator struct {
	db      *DB
	history map[int]bool
}

// NewMigrator creates a new Migrator instance.
func NewMigrator(db *DB) *Migrator {
	return &Migrator{
		db:      db,
		history: make(map[int]bool),
	}
}

// Init initializes the migration history table.
func (m *Migrator) Init() error {
	const createTableSQL = `
		CREATE TABLE IF NOT EXISTS jorm_migrations (
			version INTEGER PRIMARY KEY,
			description TEXT,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`
	_, err := m.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to initialize migration table: %w", err)
	}

	rows, err := m.db.pool.QueryContext(context.Background(), "SELECT version FROM jorm_migrations")
	if err != nil {
		return fmt.Errorf("failed to fetch migration history: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return err
		}
		m.history[version] = true
	}
	return nil
}

// Migrate executes a list of migrations that haven't been applied yet.
func (m *Migrator) Migrate(migrations ...*Migration) error {
	if err := m.Init(); err != nil {
		return err
	}

	for _, mig := range migrations {
		if m.history[mig.Version] {
			continue
		}

		err := m.db.Transaction(func(tx *Tx) error {
			if err := mig.Up(m.db); err != nil {
				return err
			}

			const insertSQL = "INSERT INTO jorm_migrations (version, description) VALUES (?, ?)"
			_, err := tx.db.Exec(insertSQL, mig.Version, mig.Description)
			return err
		})

		if err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", mig.Version, mig.Description, err)
		}

		m.history[mig.Version] = true
	}

	return nil
}

// Rollback rolls back the last applied migration.
func (m *Migrator) Rollback(mig *Migration) error {
	if !m.history[mig.Version] {
		return fmt.Errorf("migration %d not applied", mig.Version)
	}

	err := m.db.Transaction(func(tx *Tx) error {
		if err := mig.Down(m.db); err != nil {
			return err
		}

		const deleteSQL = "DELETE FROM jorm_migrations WHERE version = ?"
		_, err := tx.db.Exec(deleteSQL, mig.Version)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to rollback migration %d (%s): %w", mig.Version, mig.Description, err)
	}

	delete(m.history, mig.Version)
	return nil
}
