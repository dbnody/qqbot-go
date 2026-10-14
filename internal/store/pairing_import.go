package store

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type legacyPair struct {
	Date       string `json:"date"`
	FirstName  string `json:"first_name"`
	SecondName string `json:"second_name"`
}

// ImportLegacyPairHistory imports the Python plugin's per-group JSON history.
// Existing Go records win, so the operation is safe to repeat.
func (s *Store) ImportLegacyPairHistory(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var history map[string]legacyPair
	if err = json.Unmarshal(data, &history); err != nil {
		return 0, fmt.Errorf("decode legacy CP history: %w", err)
	}
	if history == nil {
		return 0, fmt.Errorf("decode legacy CP history: expected an object")
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT INTO cp_pairs(group_id,pair_date,first_name,second_name) VALUES(?,?,?,?) ON CONFLICT DO NOTHING`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	imported := 0
	for group, pair := range history {
		if group == "" || pair.FirstName == "" || pair.SecondName == "" {
			return 0, fmt.Errorf("invalid legacy CP history entry for group %q", group)
		}
		if _, err = time.Parse("2006-01-02", pair.Date); err != nil {
			return 0, fmt.Errorf("invalid legacy CP date for group %q: %w", group, err)
		}
		result, execErr := stmt.Exec(group, pair.Date, pair.FirstName, pair.SecondName)
		if execErr != nil {
			return 0, execErr
		}
		rows, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return 0, rowsErr
		}
		imported += int(rows)
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return imported, nil
}
