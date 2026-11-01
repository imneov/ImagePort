package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/imneov/ImagePort/internal/domain"
)

// InitializeTargets atomically saves initial locations once per database, preserving existing locations.
// It returns whether initialization added any locations; removing them later does not recreate them.
func (store *Store) InitializeTargets(initial []domain.Target) (bool, error) {
	existing, err := store.Targets()
	if err != nil {
		return false, err
	}
	transaction, err := store.db.Begin()
	if err != nil {
		return false, err
	}
	var marker string
	err = transaction.QueryRow("SELECT value FROM app_state WHERE key='initial-targets'").Scan(&marker)
	if err == nil {
		return false, transaction.Rollback()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, errors.Join(err, transaction.Rollback())
	}
	changed := false
	for _, target := range initial {
		found := false
		for _, saved := range existing {
			found = found || saved.Kind == target.Kind && saved.Address == target.Address
		}
		if found {
			continue
		}
		if err := domain.ValidateTarget(target); err != nil {
			return false, errors.Join(err, transaction.Rollback())
		}
		data, err := json.Marshal(target)
		if err != nil {
			return false, errors.Join(err, transaction.Rollback())
		}
		if _, err := transaction.Exec("INSERT INTO targets(id,data) VALUES(?,?)", target.ID, data); err != nil {
			return false, errors.Join(err, transaction.Rollback())
		}
		changed = true
	}
	if _, err := transaction.Exec("INSERT INTO app_state(key,value) VALUES('initial-targets','done')"); err != nil {
		return false, errors.Join(err, transaction.Rollback())
	}
	return changed, transaction.Commit()
}
