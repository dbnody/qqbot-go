package bot

import (
	"context"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"qqbot-go/internal/onebot"
	"strings"
	"testing"
)

func TestReviewSources(t *testing.T) {
	e := onebot.Event{Message: json.RawMessage(`[{"type":"at","data":{"qq":"all"}},{"type":"at","data":{"qq":123}},{"type":"image","data":{"url":"https://example.com/a.png","file":"opaque"}}]`)}
	target, source := reviewSources(e)
	if target != "123" || source != "https://example.com/a.png" {
		t.Fatalf("sources = %q %q", target, source)
	}
}

func TestVisualReviewSendsImageAndRedactsSecrets(t *testing.T) {
	t.Setenv("AI_API_KEY", "secret-test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/image" {
			w.Header().Set("Content-Type", "image/png")
			_ = png.Encode(w, image.NewRGBA(image.Rect(0, 0, 2, 2)))
			return
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret-test-key" {
			t.Error("missing authorization")
		}
		var body struct {
			Messages []struct {
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if len(body.Messages) != 2 || !strings.Contains(string(body.Messages[1].Content), "data:image/png;base64,") {
			t.Errorf("missing multimodal image: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"配色明快 secret-test-key"}}]}`))
	}))
	defer server.Close()
	t.Setenv("AI_API_BASE_URL", server.URL)
	got := visualReview(context.Background(), server.URL+"/image", "评价构图")
	if got != "配色明快 [已隐藏]" {
		t.Fatalf("answer = %q", got)
	}
}

func TestImageDataURLRejectsNonImages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("not an image")) }))
	defer server.Close()
	for _, source := range []string{server.URL, "file:///secret.txt"} {
		if _, err := imageDataURL(context.Background(), source); err == nil {
			t.Errorf("accepted %s", source)
		}
	}
}
