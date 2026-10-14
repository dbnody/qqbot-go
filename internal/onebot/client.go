package onebot

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	mu      sync.Mutex
	writeMu sync.Mutex
	conn    *websocket.Conn
	pending map[uint64]chan json.RawMessage
	next    atomic.Uint64
	onEvent func(Event)
}
type request struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
	Echo   uint64         `json:"echo"`
}

func NewClient() *Client { return &Client{pending: make(map[uint64]chan json.RawMessage)} }
func (c *Client) Attach(x *websocket.Conn, f func(Event)) {
	c.mu.Lock()
	old := c.conn
	for id, ch := range c.pending {
		select {
		case ch <- nil:
		default:
		}
		delete(c.pending, id)
	}
	c.conn = x
	c.onEvent = f
	c.mu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	go c.readLoop(x, f)
}
func (c *Client) readLoop(x *websocket.Conn, f func(Event)) {
	defer func() {
		_ = x.Close()
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.conn == x {
			c.conn = nil
			for id, ch := range c.pending {
				select {
				case ch <- nil:
				default:
				}
				delete(c.pending, id)
			}
		}
	}()
	for {
		_, d, e := x.ReadMessage()
		if e != nil {
			return
		}
		var v struct {
			Echo uint64 `json:"echo"`
		}
		if json.Unmarshal(d, &v) == nil && v.Echo != 0 {
			c.HandleResponse(d)
			continue
		}
		var event Event
		if event.UnmarshalJSON(d) == nil && event.PostType != "" && f != nil {
			go f(event)
		}
	}
}
func (c *Client) HandleResponse(d []byte) {
	var v struct {
		Echo uint64 `json:"echo"`
	}
	if json.Unmarshal(d, &v) != nil || v.Echo == 0 {
		return
	}
	c.mu.Lock()
	ch := c.pending[v.Echo]
	delete(c.pending, v.Echo)
	c.mu.Unlock()
	if ch != nil {
		ch <- d
	}
}
func (c *Client) Call(ctx context.Context, a string, p map[string]any) (json.RawMessage, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
	}
	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return nil, errors.New("not connected")
	}
	id := c.next.Add(1)
	ch := make(chan json.RawMessage, 1)
	c.pending[id] = ch
	x := c.conn
	c.mu.Unlock()
	d, e := json.Marshal(request{a, p, id})
	if e != nil {
		c.removePending(id)
		return nil, e
	}
	c.writeMu.Lock()
	deadline, _ := ctx.Deadline()
	_ = x.SetWriteDeadline(deadline)
	e = x.WriteMessage(websocket.TextMessage, d)
	c.writeMu.Unlock()
	if e != nil {
		c.removePending(id)
		return nil, e
	}
	select {
	case r := <-ch:
		if r == nil {
			return nil, errors.New("connection closed")
		}
		var status struct {
			RetCode *int   `json:"retcode"`
			Status  string `json:"status"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(r, &status); err != nil {
			return nil, err
		}
		if status.Status == "failed" || (status.RetCode != nil && *status.RetCode != 0) {
			if status.Message == "" {
				status.Message = "OneBot action failed"
			}
			return nil, errors.New(status.Message)
		}
		return r, nil
	case <-ctx.Done():
		c.removePending(id)
		return nil, ctx.Err()
	}
}
func (c *Client) removePending(id uint64) { c.mu.Lock(); delete(c.pending, id); c.mu.Unlock() }
func (c *Client) SendMessage(ctx context.Context, e Event, m string) error {
	p := map[string]any{`message`: []Segment{{Type: "text", Data: map[string]any{"text": m}}}}
	if e.MessageType == `group` {
		p[`group_id`] = e.GroupID
	} else if e.MessageType == `private` {
		p[`user_id`] = e.UserID
	} else {
		return errors.New("unsupported message type")
	}
	_, x := c.Call(ctx, "send_msg", p)
	return x
}
func (c *Client) CallAPI(ctx context.Context, action string, params map[string]any) (json.RawMessage, error) {
	return c.Call(ctx, action, params)
}
