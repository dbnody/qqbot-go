package store

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
)

type Store struct{ DB *sql.DB }

func Open(path string) (*Store, error) {
	if d := filepath.Dir(path); d != "." {
		if e := os.MkdirAll(d, 0755); e != nil {
			return nil, e
		}
	}
	db, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	s := &Store{db}
	if e = s.init(); e != nil {
		db.Close()
		return nil, e
	}
	return s, nil
}
func (s *Store) init() error {
	_, e := s.DB.Exec(`CREATE TABLE IF NOT EXISTS users(user_id TEXT PRIMARY KEY,nickname TEXT NOT NULL DEFAULT '',created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS sign_ins(id INTEGER PRIMARY KEY AUTOINCREMENT,user_id TEXT NOT NULL,sign_date TEXT NOT NULL,created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,UNIQUE(user_id,sign_date));
CREATE TABLE IF NOT EXISTS groups(group_id TEXT PRIMARY KEY,group_name TEXT NOT NULL DEFAULT '',created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS cultivators(user_id TEXT PRIMARY KEY,nickname TEXT NOT NULL DEFAULT '',realm_index INTEGER NOT NULL DEFAULT 0,experience INTEGER NOT NULL DEFAULT 0,spirit_stones INTEGER NOT NULL DEFAULT 0,updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE IF NOT EXISTS group_messages(id INTEGER PRIMARY KEY AUTOINCREMENT,group_id TEXT NOT NULL,user_id TEXT NOT NULL,nickname TEXT NOT NULL DEFAULT '',message_text TEXT NOT NULL DEFAULT '',created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE INDEX IF NOT EXISTS idx_group_messages_group_time ON group_messages(group_id,created_at);
CREATE INDEX IF NOT EXISTS idx_group_messages_group_user ON group_messages(group_id,user_id);
CREATE TABLE IF NOT EXISTS ai_memories(id INTEGER PRIMARY KEY AUTOINCREMENT,scope_id TEXT NOT NULL,role TEXT NOT NULL,content TEXT NOT NULL,created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE INDEX IF NOT EXISTS idx_ai_memories_scope_time ON ai_memories(scope_id,id);
CREATE TABLE IF NOT EXISTS cp_pairs(group_id TEXT NOT NULL,pair_date TEXT NOT NULL,first_name TEXT NOT NULL,second_name TEXT NOT NULL,PRIMARY KEY(group_id,pair_date));`)
	return e
}
