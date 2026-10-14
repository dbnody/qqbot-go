package onebot

import (
	"github.com/gorilla/websocket"
	"net/http"
)

type Server struct {
	Token   string
	Client  *Client
	OnEvent func(Event)
}

func NewServer(t string, c *Client, f func(Event)) *Server { return &Server{t, c, f} }

var upgrader = websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.Token != "" && r.Header.Get("Authorization") != "Bearer "+s.Token {
		http.Error(w, "unauthorized", 401)
		return
	}
	c, e := upgrader.Upgrade(w, r, nil)
	if e != nil {
		return
	}
	s.Client.Attach(c, s.OnEvent)
}
