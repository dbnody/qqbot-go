package store

import (
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })
	return s
}

func TestSignInIsUniquePerDay(t *testing.T) {
	s := testStore(t)
	ok, reward, total, err := s.SignIn("user-1", "alice")
	if err != nil || !ok || reward < 5 || reward > 20 || total != 1 {
		t.Fatalf("first sign-in = %v, %d, %d, %v", ok, reward, total, err)
	}
	ok, reward, total, err = s.SignIn("user-1", "alice")
	if err != nil || ok || reward != 0 || total != 0 {
		t.Fatalf("duplicate sign-in = %v, %d, %d, %v", ok, reward, total, err)
	}
}

func TestCultivationAndBreakthrough(t *testing.T) {
	s := testStore(t)
	p, gain, err := s.Cultivate("user-2", "bob")
	if err != nil || gain < 8 || gain > 24 || p.Experience != gain || p.Stones != 1 || p.Name != "炼气期" {
		t.Fatalf("cultivation = %+v gain=%d err=%v", p, gain, err)
	}
	_, err = s.DB.Exec("UPDATE cultivators SET experience=100 WHERE user_id=?", "user-2")
	if err != nil {
		t.Fatal(err)
	}
	old, next, ok, max, _, err := s.Breakthrough("user-2", "bob")
	if err != nil || !ok || max || old != "炼气期" || next != "筑基期" {
		t.Fatalf("breakthrough = %q -> %q ok=%v max=%v err=%v", old, next, ok, max, err)
	}
	p, err = s.Profile("user-2", "bob")
	if err != nil || p.Name != "筑基期" || p.Experience != 0 {
		t.Fatalf("profile after breakthrough = %+v err=%v", p, err)
	}
}
