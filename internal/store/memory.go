package store

import "strings"

type Memory struct {
	Role    string
	Content string
}

func (s *Store) LoadMemory(scope string, limit int) ([]Memory, error) {
	if limit < 0 {
		limit = 0
	}
	if limit > 20 {
		limit = 20
	}
	rows, e := s.DB.Query(`SELECT role,content FROM (SELECT id,role,content FROM ai_memories WHERE scope_id=? ORDER BY id DESC LIMIT ?) ORDER BY id ASC`, scope, limit)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Memory{}
	for rows.Next() {
		var m Memory
		if e = rows.Scan(&m.Role, &m.Content); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Store) AppendMemory(scope, role, content string) error {
	if role != "user" && role != "assistant" {
		return nil
	}
	content = strings.TrimSpace(content)
	if r := []rune(content); len(r) > 4000 {
		content = string(r[:4000])
	}
	if content == "" {
		return nil
	}
	if _, e := s.DB.Exec(`INSERT INTO ai_memories(scope_id,role,content) VALUES(?,?,?)`, scope, role, content); e != nil {
		return e
	}
	_, e := s.DB.Exec(`DELETE FROM ai_memories WHERE scope_id=? AND id NOT IN(SELECT id FROM ai_memories WHERE scope_id=? ORDER BY id DESC LIMIT 40)`, scope, scope)
	return e
}
func (s *Store) ClearMemory(scope string) error {
	_, e := s.DB.Exec(`DELETE FROM ai_memories WHERE scope_id=?`, scope)
	return e
}
