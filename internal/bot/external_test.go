package bot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"qqbot-go/internal/onebot"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func mockExternalHTTP(t *testing.T, fn func(*http.Request) string) {
	t.Helper()
	old := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fn(r))), Header: http.Header{}, Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = old })
}
func TestMusicSelectsArtistAndSendsCards(t *testing.T) {
	mockExternalHTTP(t, func(r *http.Request) string {
		if r.URL.Host == "c.y.qq.com" {
			return `{"data":{"song":{"list":[{"songid":1,"songmid":"cover","songname":"晴天","singer":[{"name":"翻唱"}]},{"songid":2,"songmid":"original","songname":"晴天","singer":[{"name":"周杰伦"}]}]}}}`
		}
		return `{"result":{"songs":[{"id":3,"name":"晴天","artists":[{"name":"周杰伦"}]}]}}`
	})
	h, calls := mockBot(t, func(c mockCall) (any, int) { return nil, 0 })
	e := eventJSON(t, `{"message_type":"group","group_id":55,"user_id":3}`)
	if got := h.musicCommand(e, "周杰伦 晴天"); got != "" {
		t.Fatal(got)
	}
	got := calls()
	segs := onebot.ParseMessage(got[0].Params["message"])
	if len(segs) != 3 || segs[1].Data["id"] != "2" || segs[2].Data["id"] != "3" {
		t.Fatal(segs)
	}
}
func TestBiliResolvesCaseSensitiveBVAndSelectedPage(t *testing.T) {
	mockExternalHTTP(t, func(r *http.Request) string {
		if r.URL.Path == "/x/web-interface/view" {
			if r.URL.Query().Get("bvid") != "BV1ab2CD3ef4" {
				t.Error("BV case changed")
			}
			return `{"code":0,"data":{"bvid":"BV1ab2CD3ef4","title":"测试视频","duration":10,"pages":[{"cid":1},{"cid":2}]}}`
		}
		if r.URL.Query().Get("cid") != "2" {
			t.Error("selected page not used")
		}
		return `{"code":0,"data":{"durl":[{"url":"https://example.com/video.mp4","size":20}]}}`
	})
	v, err := resolveBili(context.Background(), "https://bilibili.com/video/BV1ab2CD3ef4?p=2")
	if err != nil || v.CID != 2 || v.Title != "测试视频" {
		t.Fatalf("video: %v %v", v, err)
	}
}
func TestBiliDownloadFailureRemovesPartialFile(t *testing.T) {
	t.Chdir(t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("denied"))
	}))
	defer server.Close()
	_, err := downloadBili(context.Background(), biliVideo{URL: server.URL})
	if err == nil {
		t.Fatal("accepted forbidden download")
	}
	files, _ := filepath.Glob("data/bilibili/*")
	if len(files) != 0 {
		t.Fatal("partial files retained")
	}
}
func TestBiliShareCardAndVideoSend(t *testing.T) {
	t.Chdir(t.TempDir())
	mockExternalHTTP(t, func(r *http.Request) string {
		switch r.URL.Path {
		case "/x/web-interface/view":
			return `{"code":0,"data":{"bvid":"BV1ab2CD3ef4","title":"分享测试","duration":10,"pages":[{"cid":1}]}}`
		case "/x/player/playurl":
			return `{"code":0,"data":{"durl":[{"url":"https://example.com/mock.mp4","size":4}]}}`
		default:
			return "mp4!"
		}
	})
	h, calls := mockBot(t, func(c mockCall) (any, int) { return map[string]any{"message_id": 222}, 0 })
	e := eventJSON(t, `{"message_type":"group","group_id":3,"user_id":2,"self_id":99,"message":[{"type":"share","data":{"url":"https%3A%2F%2Fbilibili.com%2Fvideo%2FBV1ab2CD3ef4"}}]}`)
	if got := h.handleBilibili(e, ""); got != "" {
		t.Fatal(got)
	}
	all := calls()
	if len(all) != 3 || all[1].Action != "send_group_msg" || all[2].Action != "delete_msg" {
		t.Fatal(all)
	}
	var segments []onebot.Segment
	if err := json.Unmarshal(all[1].Params["message"], &segments); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(segments[1].Data["file"].(string), "file:///") {
		t.Fatal("video path not absolute URL")
	}
	files, _ := os.ReadDir("data/bilibili")
	if len(files) > 0 {
		t.Fatal("video not cleaned up")
	}
}
