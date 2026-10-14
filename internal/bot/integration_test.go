package bot

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/websocket"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"qqbot-go/internal/onebot"
	"qqbot-go/internal/store"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockCall struct {
	Action string                     `json:"action"`
	Params map[string]json.RawMessage `json:"params"`
	Echo   uint64                     `json:"echo"`
}

func mockBot(t *testing.T, respond func(mockCall) (any, int)) (*Handler, func() []mockCall) {
	t.Helper()
	client := onebot.NewClient()
	s, err := store.Open(filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Close() })
	h := NewHandler(client, "/", s)
	ready := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		client.Attach(conn, nil)
		close(ready)
	}))
	t.Cleanup(server.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	<-ready
	var mu sync.Mutex
	var calls []mockCall
	go func() {
		for {
			var req mockCall
			if err := conn.ReadJSON(&req); err != nil {
				return
			}
			mu.Lock()
			calls = append(calls, req)
			mu.Unlock()
			data, code := respond(req)
			if err := conn.WriteJSON(map[string]any{"echo": req.Echo, "status": map[bool]string{true: "ok", false: "failed"}[code == 0], "retcode": code, "data": data}); err != nil {
				return
			}
		}
	}()
	return h, func() []mockCall { mu.Lock(); defer mu.Unlock(); return append([]mockCall(nil), calls...) }
}
func eventJSON(t *testing.T, s string) onebot.Event {
	t.Helper()
	var e onebot.Event
	if err := json.Unmarshal([]byte(s), &e); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestAIChatImageMemoryNoticeAndCooldown(t *testing.T) {
	t.Setenv("AI_API_KEY", "integration-secret")
	var request map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/image" {
			_ = png.Encode(w, image.NewRGBA(image.Rect(0, 0, 2, 2)))
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": []any{map[string]string{"text": "看到了图片"}}}}}})
	}))
	defer api.Close()
	t.Setenv("AI_API_BASE_URL", api.URL)
	t.Setenv("AI_MODEL", "kimi-test")
	h, calls := mockBot(t, func(c mockCall) (any, int) { return map[string]any{"message_id": 1001}, 0 })
	e := eventJSON(t, `{"post_type":"message","message_type":"group","group_id":8484,"user_id":25,"self_id":99,"message_id":7,"sender":{"nickname":"甲"},"message":[{"type":"at","data":{"qq":"99"}},{"type":"text","data":{"text":"看这张图片"}},{"type":"image","data":{"url":"`+api.URL+`/image"}}]}`)
	h.HandleEvent(e)
	got := calls()
	if len(got) != 3 || got[0].Action != "send_msg" || got[1].Action != "delete_msg" || got[2].Action != "send_msg" {
		t.Fatalf("notice lifecycle: %+v", got)
	}
	if string(got[1].Params["message_id"]) != "1001" {
		t.Fatal("deleted wrong message")
	}
	if request["temperature"] != 0.6 || request["thinking"] == nil {
		t.Fatal("Kimi compatibility missing")
	}
	messages := request["messages"].([]any)
	latest := messages[len(messages)-1].(map[string]any)["content"].([]any)
	if !strings.Contains(latest[1].(map[string]any)["image_url"].(map[string]any)["url"].(string), "data:image/png;base64,") {
		t.Fatal("image not included")
	}
	memory, err := h.Store.LoadMemory("group:8484", 8)
	if err != nil || len(memory) != 2 {
		t.Fatalf("memory: %v %v", memory, err)
	}
	other, _ := h.Store.LoadMemory("private:25", 8)
	if len(other) != 0 {
		t.Fatal("memory isolation failed")
	}
	h.HandleEvent(e)
	if len(calls()) != 3 {
		t.Fatal("cooldown did not suppress duplicate")
	}
}

func TestModerationFailureAndProtectedRoles(t *testing.T) {
	t.Setenv("QQBOT_FORBIDDEN_WORDS", "bad")
	t.Setenv("QQBOT_FORBIDDEN_MUTE_SECONDS", "60")
	h, calls := mockBot(t, func(c mockCall) (any, int) {
		if c.Action == "get_group_member_info" {
			return map[string]any{"role": "admin"}, 0
		}
		if c.Action == "delete_msg" {
			return nil, 100
		}
		return nil, 0
	})
	e := eventJSON(t, `{"post_type":"message","message_type":"group","group_id":3,"user_id":2,"self_id":99,"message_id":8,"sender":{"role":"member"},"message":"bad"}`)
	if got := h.moderate(e, "bad", "3", "2"); got != "" {
		t.Fatalf("false success: %s", got)
	}
	got := calls()
	if len(got) != 2 || got[1].Action != "delete_msg" {
		t.Fatalf("calls: %v", got)
	}
	e.Raw["sender"] = map[string]any{"role": "owner"}
	h.moderate(e, "bad", "3", "2")
	if len(calls()) != 2 {
		t.Fatal("owner was moderated")
	}
}

func TestGroupBanParsesNumericTargetDuration(t *testing.T) {
	h, calls := mockBot(t, func(c mockCall) (any, int) {
		if c.Action == "get_group_member_info" {
			role := "member"
			if string(c.Params["user_id"]) == "99" {
				role = "admin"
			}
			return map[string]any{"role": role}, 0
		}
		return nil, 0
	})
	e := eventJSON(t, `{"post_type":"message","message_type":"group","group_id":3,"user_id":2,"self_id":99,"sender":{"role":"admin"},"message":"/ban 123456 2小时"}`)
	got := h.groupAdmin("ban", "123456 2小时", e)
	if !strings.Contains(got, "120 分钟") {
		t.Fatal(got)
	}
	all := calls()
	if string(all[len(all)-1].Params["duration"]) != "7200" {
		t.Fatal("QQ number used as duration")
	}
}

func TestTarotCommandSendsBase64AndRemovesFile(t *testing.T) {
	t.Chdir(t.TempDir())
	h, calls := mockBot(t, func(c mockCall) (any, int) { return nil, 0 })
	e := eventJSON(t, `{"post_type":"message","message_type":"private","user_id":2,"self_id":99,"message":"/塔罗 工作"}`)
	h.HandleEvent(e)
	got := calls()
	if len(got) != 1 {
		t.Fatalf("calls: %v", got)
	}
	segments := onebot.ParseMessage(got[0].Params["message"])
	if len(segments) != 2 || !strings.Contains(segments[0].Data["text"].(string), "关键词") {
		t.Fatalf("message: %v", segments)
	}
	file := segments[1].Data["file"].(string)
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(file, "base64://"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width != 720 || cfg.Height != 960 {
		t.Fatalf("PNG: %v %v", cfg, err)
	}
	files, _ := filepath.Glob("data/tarot/*.png")
	if len(files) != 0 {
		t.Fatal("tarot files leaked")
	}
}

func TestRecallFallbackAndSelfFiltering(t *testing.T) {
	t.Setenv("QQBOT_ADMIN_IDS", "12345")
	t.Setenv("SUPERUSERS", "")
	h, calls := mockBot(t, func(c mockCall) (any, int) {
		if c.Action == "get_msg" {
			return map[string]any{"message": `[CQ:image,file=old,url=https://example.com/a.png]`}, 0
		}
		return nil, 0
	})
	e := eventJSON(t, `{"post_type":"notice","notice_type":"friend_recall","user_id":2,"self_id":99,"message_id":834834}`)
	h.HandleEvent(e)
	got := calls()
	if len(got) != 2 || got[0].Action != "get_msg" || got[1].Action != "send_private_msg" {
		t.Fatal(got)
	}
	segs := onebot.ParseMessage(got[1].Params["message"])
	if segs[1].Data["file"] != "https://example.com/a.png" {
		t.Fatal("media not forwarded")
	}
	e.UserID = json.RawMessage("99")
	h.HandleEvent(e)
	if len(calls()) != 2 {
		t.Fatal("self recall forwarded")
	}
}

func TestCPReturnsPersistedPairWithoutOneBotLookup(t *testing.T) {
	h, calls := mockBot(t, func(c mockCall) (any, int) { return nil, 0 })
	_, err := h.Store.SaveDailyPair("4", time.Now().Format("2006-01-02"), "甲", "乙")
	if err != nil {
		t.Fatal(err)
	}
	e := eventJSON(t, `{"message_type":"group","group_id":4,"user_id":2,"self_id":99}`)
	if got := h.pairing(e); !strings.Contains(got, "甲 × 乙") {
		t.Fatal(got)
	}
	if len(calls()) != 0 {
		t.Fatal("existing pair queried members")
	}
}
