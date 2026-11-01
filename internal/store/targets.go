package store

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/imneov/ImagePort/internal/domain"
)

// Targets returns detached saved locations in creation order, without credentials.
func (store *Store) Targets() ([]domain.Target, error) {
	rows, err := store.db.Query("SELECT data FROM targets ORDER BY rowid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Target
	for rows.Next() {
		var data []byte
		var target domain.Target
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &target); err != nil {
			return nil, err
		}
		result = append(result, target)
	}
	return result, rows.Err()
}

// SaveTarget validates and persists a location; it never creates a remote project.
func (store *Store) SaveTarget(target domain.Target) error {
	if target.ID == "" {
		return errors.New("The target has no identifier")
	}
	if err := domain.ValidateTarget(target); err != nil {
		return err
	}
	data, err := json.Marshal(target)
	if err != nil {
		return err
	}
	_, err = store.db.Exec("INSERT INTO targets(id,data) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET data=excluded.data", target.ID, data)
	return err
}

// DeleteTarget removes configuration only, retaining every image, file and task.
func (store *Store) DeleteTarget(id string) error {
	_, err := store.db.Exec("DELETE FROM targets WHERE id=?", id)
	return err
}

// DeleteProfile removes account metadata after its secret has been explicitly removed.
func (store *Store) DeleteProfile(id string) error {
	_, err := store.db.Exec("DELETE FROM profiles WHERE id=?", id)
	return err
}

// migrateTargets separates legacy connection rows into locations and access accounts.
func (store *Store) migrateTargets() error {
	profiles, err := store.Profiles()
	if err != nil {
		return err
	}
	transaction, err := store.db.Begin()
	if err != nil {
		return err
	}
	locations := make(map[string]string)
	for _, profile := range profiles {
		scheme := "https://"
		if profile.HTTP {
			scheme = "http://"
		}
		address := scheme + profile.Host
		if locations[address] == "" {
			locations[address] = profile.ID
			data, marshalErr := json.Marshal(domain.Target{ID: profile.ID, Name: profile.Name, Kind: "registry", Address: address})
			if marshalErr != nil {
				return errors.Join(marshalErr, transaction.Rollback())
			}
			if _, err := transaction.Exec("INSERT INTO targets(id,data) VALUES(?,?)", profile.ID, data); err != nil {
				return errors.Join(err, transaction.Rollback())
			}
		}
		profile.TargetID = locations[address]
		data, err := json.Marshal(profile)
		if err != nil {
			return errors.Join(err, transaction.Rollback())
		}
		if _, err := transaction.Exec("UPDATE profiles SET data=? WHERE id=?", data, profile.ID); err != nil {
			return errors.Join(fmt.Errorf("migrate access account: %w", err), transaction.Rollback())
		}
	}
	if _, err := transaction.Exec("PRAGMA user_version=2"); err != nil {
		return errors.Join(err, transaction.Rollback())
	}
	return transaction.Commit()
}
