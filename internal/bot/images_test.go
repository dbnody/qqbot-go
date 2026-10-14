package bot

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"qqbot-go/internal/store"
)

func TestMakeMemeWritesPNG(t *testing.T) {
	t.Chdir(t.TempDir())
	h := &Handler{}
	path, err := h.makeMeme("狗头 今天继续写代码")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode generated meme: %v", err)
	}
	if got := img.Bounds().Size(); got.X != 900 || got.Y != 650 {
		t.Fatalf("meme size = %v, want 900x650", got)
	}
}

func TestMakeMemeFallsBackForUnknownTemplate(t *testing.T) {
	t.Chdir(t.TempDir())
	h := &Handler{}
	if path, err := h.makeMeme("不存在 文字"); err != nil {
		t.Fatalf("classic fallback: %v", err)
	} else {
		_ = os.Remove(path)
	}
}

func TestWordCloudUsesGroupMessageCounts(t *testing.T) {
	t.Chdir(t.TempDir())
	s, err := store.Open(filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })
	if err := s.RecordMessage("group-1", "user-1", "alice", "今天学习 Go 编程"); err != nil {
		t.Fatal(err)
	}
	h := &Handler{Store: s}
	path, err := h.wordCloud("group-1", 30)
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("expected word cloud image")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := png.Decode(f); err != nil {
		t.Fatalf("decode generated word cloud: %v", err)
	}
}
