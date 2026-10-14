package store

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"unicode/utf8"
)

func TestConcurrentSignInUniqueAndMemoryUTF8(t *testing.T) {
	s := testStore(t)
	var wg sync.WaitGroup
	var winners atomic.Int32
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, _, _, err := s.SignIn("same", "甲")
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
		t.Fatalf("winners=%d", winners.Load())
	}
	if err := s.AppendMemory("scope", "user", strings.Repeat("中文", 3000)); err != nil {
		t.Fatal(err)
	}
	mem, err := s.LoadMemory("scope", 1)
	if err != nil || len(mem) != 1 || !utf8.ValidString(mem[0].Content) || len([]rune(mem[0].Content)) != 4000 {
		t.Fatal("UTF8 memory truncated incorrectly")
	}
}
