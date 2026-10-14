package store

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDailyPairSurvivesReopenAndConcurrentClaims(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairs.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := s.SaveDailyPair("g", "2026-10-14", "Alice", "Bob")
			if err != nil {
				t.Error(err)
			}
			if ok {
				winners.Add(1)
			}
		}()
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Fatalf("claims = %d", winners.Load())
	}
	if err = s.DB.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	first, second, err := s.DailyPair("g", "2026-10-14")
	if err != nil || first != "Alice" || second != "Bob" {
		t.Fatalf("persisted pair = %q %q %v", first, second, err)
	}
	ok, err := s.SaveDailyPair("g", "2026-10-15", "Carol", "Dave")
	if err != nil || !ok {
		t.Fatalf("next day = %v %v", ok, err)
	}
	first, _, err = s.DailyPair("other", "2026-10-14")
	if err != nil || first != "" {
		t.Fatal("group isolation failed")
	}
}

func TestImportLegacyPairHistoryIsIdempotentAndTransactional(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "pairs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	path := filepath.Join(t.TempDir(), "cp_history.json")
	if err = os.WriteFile(path, []byte(`{"123":{"date":"2026-10-14","first_id":1,"second_id":2,"first_name":"Alice","second_name":"Bob"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	count, err := s.ImportLegacyPairHistory(path)
	if err != nil || count != 1 {
		t.Fatalf("first import = %d, %v", count, err)
	}
	count, err = s.ImportLegacyPairHistory(path)
	if err != nil || count != 0 {
		t.Fatalf("repeat import = %d, %v", count, err)
	}
	first, second, err := s.DailyPair("123", "2026-10-14")
	if err != nil || first != "Alice" || second != "Bob" {
		t.Fatalf("imported pair = %q %q, %v", first, second, err)
	}

	invalid := filepath.Join(t.TempDir(), "invalid.json")
	if err = os.WriteFile(invalid, []byte(`{"456":{"date":"bad","first_name":"Carol","second_name":"Dave"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ImportLegacyPairHistory(invalid); err == nil {
		t.Fatal("invalid legacy entry accepted")
	}
	first, _, err = s.DailyPair("456", "2026-10-14")
	if err != nil || first != "" {
		t.Fatalf("invalid import was partially committed: %q, %v", first, err)
	}
}
