package store

import "testing"

func TestWordCountsFiltersCommandsURLsAndOtherGroups(t *testing.T) {
	s := testStore(t)
	for _, text := range []string{"学习 Go go", "/meme 狗头 文字", "https://example.com/ignored [CQ:image,file=test]"} {
		if err := s.RecordMessage("group-1", "alice", "Alice", text); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RecordMessage("group-2", "bob", "Bob", "OtherGroup"); err != nil {
		t.Fatal(err)
	}
	counts, err := s.WordCounts("group-1", 30)
	if err != nil {
		t.Fatal(err)
	}
	if counts["学习"] != 1 || counts["go"] != 2 {
		t.Fatalf("unexpected counts: %#v", counts)
	}
	if len(counts) != 2 {
		t.Fatalf("commands, URLs or other groups leaked into counts: %#v", counts)
	}
}
