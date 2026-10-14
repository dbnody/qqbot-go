package bot

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestMusicRequestAndCardSegments(t *testing.T) {
	for _, tc := range []struct{ input, query, platform string }{
		{"qq 歌手：周杰伦 歌名：晴天", "周杰伦 晴天", "qq"},
		{"网易云 晴天 - 周杰伦", "晴天 周杰伦", "163"},
		{"歌名：晴天", "晴天", "both"},
	} {
		q, p := parseMusicRequest(tc.input)
		if q != tc.query || p != tc.platform {
			t.Fatalf("%q => %q %q", tc.input, q, p)
		}
	}
	segs := musicSegments("结果", []songResult{{Platform: "qq", ID: 123}, {Platform: "163", ID: 456}})
	if len(segs) != 3 || segs[1]["type"] != "music" || segs[2]["data"].(map[string]any)["id"] != "456" {
		t.Fatalf("segments = %#v", segs)
	}
}

func TestRecallMediaUsesURLAndCacheIsBounded(t *testing.T) {
	segments := recallSegments(json.RawMessage(`[{"type":"image","data":{"file":"opaque","url":"https://example.com/a.png"}},{"type":"text","data":{"text":"hello"}}]`))
	if segments[0]["data"].(map[string]any)["file"] != "https://example.com/a.png" {
		t.Fatal("media URL not substituted")
	}
	recentMessages.Lock()
	recentMessages.items = make(map[string]json.RawMessage)
	recentMessages.order = nil
	recentMessages.Unlock()
	t.Cleanup(func() {
		recentMessages.Lock()
		recentMessages.items = make(map[string]json.RawMessage)
		recentMessages.order = nil
		recentMessages.Unlock()
	})
	for i := 0; i < 2001; i++ {
		cacheRecallMessage(fmt.Sprint(i), json.RawMessage(`"hello"`))
	}
	recentMessages.Lock()
	defer recentMessages.Unlock()
	if len(recentMessages.items) != 2000 {
		t.Fatalf("cache size = %d", len(recentMessages.items))
	}
	if _, ok := recentMessages.items["0"]; ok {
		t.Fatal("oldest entry retained")
	}
}
