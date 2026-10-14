package onebot_test

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"qqbot-go/internal/onebot"
	"strings"
	"testing"
	"time"
)

type wireCall struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
	Echo   uint64         `json:"echo"`
}

func TestServerAuthEchoFailuresAndDisconnect(t *testing.T) {
	client := onebot.NewClient()
	events := make(chan onebot.Event, 1)
	server := httptest.NewServer(onebot.NewServer("test-token", client, func(e onebot.Event) { events <- e }))
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	if conn != nil {
		conn.Close()
	}
	if err == nil || resp.StatusCode != 401 {
		t.Fatal("unauthenticated connection accepted")
	}
	header := http.Header{"Authorization": []string{"Bearer test-token"}}
	conn, _, err = websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]any{"post_type": "message", "message_type": "private", "self_id": 99, "user_id": 2, "message": "hello"})
	select {
	case e := <-events:
		if e.PlainText() != "hello" {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("event not delivered")
	}
	type result struct {
		raw json.RawMessage
		err error
	}
	output := make(chan result, 2)
	for _, action := range []string{"first", "second"} {
		go func(a string) {
			ctx, c := context.WithTimeout(context.Background(), time.Second)
			defer c()
			raw, err := client.Call(ctx, a, nil)
			output <- result{raw, err}
		}(action)
	}
	var first, second wireCall
	if err = conn.ReadJSON(&first); err != nil {
		t.Fatal(err)
	}
	if err = conn.ReadJSON(&second); err != nil {
		t.Fatal(err)
	}
	_ = conn.WriteJSON(map[string]any{"echo": second.Echo, "retcode": 0, "data": second.Action})
	_ = conn.WriteJSON(map[string]any{"echo": first.Echo, "retcode": 0, "data": first.Action})
	for i := 0; i < 2; i++ {
		if r := <-output; r.err != nil {
			t.Fatal(r.err)
		}
	}
	go func() { raw, err := client.Call(context.Background(), "failed", nil); output <- result{raw, err} }()
	var req wireCall
	_ = conn.ReadJSON(&req)
	_ = conn.WriteJSON(map[string]any{"echo": req.Echo, "status": "failed"})
	if r := <-output; r.err == nil {
		t.Fatal("failed status ignored")
	}
	go func() { raw, err := client.Call(context.Background(), "pending", nil); output <- result{raw, err} }()
	_ = conn.ReadJSON(&req)
	_ = conn.Close()
	select {
	case r := <-output:
		if r.err == nil {
			t.Fatal("disconnect not reported")
		}
	case <-time.After(time.Second):
		t.Fatal("pending call not released")
	}
}
