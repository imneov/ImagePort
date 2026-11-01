// Package store persists non-secret application metadata in SQLite.
package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/imneov/ImagePort/internal/domain"
	_ "modernc.org/sqlite" // Register the pure-Go SQLite driver.
)

// Store owns one serialized database connection and short metadata transactions.
type Store struct{ db *sql.DB }

// ErrProfileNotFound distinguishes a removed profile from database I/O failures.
var ErrProfileNotFound = errors.New("Registry profile not found; add the connection again")

// Open initializes path, applying the versioned metadata schema.
func Open(path string) (*Store, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open task database: %w", err)
	}
	database.SetMaxOpenConns(1)
	instance := &Store{db: database}
	if err = instance.migrate(); err != nil {
		return nil, errors.Join(err, database.Close())
	}
	return instance, nil
}

// migrate rejects future schemas rather than overwriting a newer application's DB.
func (store *Store) migrate() error {
	var version int
	if err := store.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version > 2 {
		return errors.New("The database was created by a newer client; upgrade ImagePort")
	}
	_, err := store.db.Exec(`PRAGMA busy_timeout=5000;
	PRAGMA journal_mode=WAL;
	CREATE TABLE IF NOT EXISTS profiles (id TEXT PRIMARY KEY, data BLOB NOT NULL);
	CREATE TABLE IF NOT EXISTS app_state (key TEXT PRIMARY KEY, value TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS targets (id TEXT PRIMARY KEY, data BLOB NOT NULL);
	CREATE TABLE IF NOT EXISTS tasks (id TEXT PRIMARY KEY, data BLOB NOT NULL);
	CREATE TABLE IF NOT EXISTS events (sequence INTEGER PRIMARY KEY AUTOINCREMENT,
	 task_id TEXT NOT NULL, time TEXT NOT NULL, state TEXT NOT NULL, stage TEXT NOT NULL);
	`)
	if err != nil || version == 2 {
		return err
	}
	return store.migrateTargets()
}

// Close releases the database connection after task shutdown.
func (store *Store) Close() error { return store.db.Close() }

// SaveTask commits a complete snapshot and a key event atomically.
func (store *Store) SaveTask(task domain.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("encode task: %w", err)
	}
	transaction, err := store.db.Begin()
	if err != nil {
		return err
	}
	_, err = transaction.Exec("INSERT OR REPLACE INTO tasks(id,data) VALUES(?,?)", task.ID, data)
	if err == nil {
		_, err = transaction.Exec("INSERT INTO events(task_id,time,state,stage) VALUES(?,?,?,?)",
			task.ID, time.Now().UTC().Format(time.RFC3339Nano), string(task.State), task.Current().Stage)
	}
	if err != nil {
		return errors.Join(err, transaction.Rollback())
	}
	return transaction.Commit()
}

// Tasks returns detached task snapshots, newest first.
func (store *Store) Tasks() ([]domain.Task, error) {
	rows, err := store.db.Query("SELECT data FROM tasks ORDER BY rowid DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []domain.Task
	for rows.Next() {
		var data []byte
		var task domain.Task
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// SaveProfile upserts public metadata; profile has no password field by design.
func (store *Store) SaveProfile(profile domain.Profile) error {
	if err := domain.ValidateProfile(profile); err != nil {
		return err
	}
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	_, err = store.db.Exec("INSERT OR REPLACE INTO profiles(id,data) VALUES(?,?)", profile.ID, data)
	return err
}

// Profiles returns all saved connections in display order.
func (store *Store) Profiles() ([]domain.Profile, error) {
	rows, err := store.db.Query("SELECT data FROM profiles ORDER BY rowid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var profiles []domain.Profile
	for rows.Next() {
		var data []byte
		var profile domain.Profile
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &profile); err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	return profiles, rows.Err()
}

// Profile loads one saved connection by id, returning an error if it was removed.
func (store *Store) Profile(id string) (domain.Profile, error) {
	var profile domain.Profile
	var data []byte
	if err := store.db.QueryRow("SELECT data FROM profiles WHERE id=?", id).Scan(&data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return profile, ErrProfileNotFound
		}
		return profile, fmt.Errorf("read registry profile: %w", err)
	}
	return profile, json.Unmarshal(data, &profile)
}
