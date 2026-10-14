package bot

import (
	"strings"
	"testing"
)

func TestCatalogCompletenessAndQuoteCycle(t *testing.T) {
	if len(catalog.Quotes) != 65 || len(catalog.Origins) != 12 || len(catalog.Talents) != 8 || len(catalog.Encounters) != 7 || len(catalog.Endings) != 8 || len(catalog.Stories) != 10 || len(catalog.Riddles) != 5 || len(catalog.Songs) != 5 || len(catalog.Idioms) != 35 || len(tarotDeck) != 22 {
		t.Fatalf("catalog sizes: %d %d %d %d %d %d %d %d %d", len(catalog.Quotes), len(catalog.Origins), len(catalog.Talents), len(catalog.Encounters), len(catalog.Endings), len(catalog.Stories), len(catalog.Riddles), len(catalog.Songs), len(catalog.Idioms))
	}
	seen := map[string]bool{}
	for range catalog.Quotes {
		q := nextQuote("test-cycle")
		if seen[q] {
			t.Fatal("repeated quote before exhaustion")
		}
		seen[q] = true
	}
	if nextQuote("test-cycle") == "" {
		t.Fatal("cycle failed")
	}
}
func TestFortuneMatchesPythonFixtures(t *testing.T) {
	for _, tc := range []struct{ scope, score, color, number string }{{"123", "大吉（77分）", "宝蓝色", "69"}, {"group:42:123", "中吉（91分）", "紫色", "55"}, {"999999999", "中吉（66分）", "暖黄色", "15"}} {
		got := fortuneForDate("2026-10-14", tc.scope, "用户")
		for _, part := range []string{tc.score, tc.color, "幸运数字：" + tc.number} {
			if !strings.Contains(got, part) {
				t.Fatalf("%s missing %s: %s", tc.scope, part, got)
			}
		}
	}
}
func TestFAQMentionBoundaryAndPunctuation(t *testing.T) {
	e := eventJSON(t, `{"message_type":"group","self_id":99,"message":"你是什么模型？"}`)
	if faqForEvent(e, e.PlainText()) != "" {
		t.Fatal("unmentioned model question answered")
	}
	e = eventJSON(t, `{"message_type":"group","self_id":99,"message":"[CQ:at,qq=99]你是什么模型？"}`)
	if faqForEvent(e, e.PlainText()) == "" {
		t.Fatal("mentioned model question ignored")
	}
	if faqForEvent(e, "谢谢！") == "" {
		t.Fatal("punctuation missing")
	}
}
