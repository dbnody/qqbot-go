package store

import "database/sql"

func (s *Store) DailyPair(group, date string) (first, second string, err error) {
	err = s.DB.QueryRow(`SELECT first_name,second_name FROM cp_pairs WHERE group_id=? AND pair_date=?`, group, date).Scan(&first, &second)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// The unique key arbitrates concurrent requests, including separate processes.
func (s *Store) SaveDailyPair(group, date, first, second string) (bool, error) {
	result, err := s.DB.Exec(`INSERT INTO cp_pairs(group_id,pair_date,first_name,second_name) VALUES(?,?,?,?) ON CONFLICT DO NOTHING`, group, date, first, second)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
