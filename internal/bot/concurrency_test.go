package bot

import (
	"encoding/json"
	"fmt"
	"qqbot-go/internal/onebot"
	"strings"
	"sync"
	"testing"
)

func TestConcurrentBlackjackJoinAndStand(t *testing.T) {
	h := &Handler{Prefix: "/"}
	e := onebot.Event{MessageType: "group", GroupID: json.RawMessage("787878"), UserID: json.RawMessage("1")}
	h.fullGameCommand("21点", "创建", e, "房主")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p := e
			p.UserID = json.RawMessage(fmt.Sprint(i + 1))
			h.fullGameCommand("21点", "加入", p, fmt.Sprint(i+1))
		}(i)
	}
	wg.Wait()
	gamesMu.Lock()
	g := blackjackGames[gameKey(e)]
	if len(g.Players) != 6 {
		t.Fatalf("players: %d", len(g.Players))
	}
	current := g.Players[0]
	gamesMu.Unlock()
	h.fullGameCommand("21点", "开始", e, "房主")
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := e
			p.UserID = json.RawMessage(current)
			h.fullGameCommand("21点", "停牌", p, current)
		}()
	}
	wg.Wait()
	gamesMu.Lock()
	defer gamesMu.Unlock()
	if g.Turn != 1 {
		t.Fatalf("duplicate stand advanced %d turns", g.Turn)
	}
	delete(blackjackGames, gameKey(e))
}
func TestRouletteFinalShotBansAndReleasesRoom(t *testing.T) {
	h, calls := mockBot(t, func(c mockCall) (any, int) { return nil, 0 })
	e := eventJSON(t, `{"message_type":"group","group_id":797979,"user_id":1}`)
	gamesMu.Lock()
	rouletteGames[gameKey(e)] = &rouletteState{Owner: "1", Players: []string{"1"}, Names: map[string]string{"1": "玩家"}, Bullet: 5, Chamber: 5, Started: true}
	gamesMu.Unlock()
	if got := h.fullGameCommand("轮盘赌", "扣扳机", e, "玩家"); !strings.Contains(got, "已禁言 60 秒") {
		t.Fatal(got)
	}
	if len(calls()) != 1 || calls()[0].Action != "set_group_ban" {
		t.Fatal(calls())
	}
	gamesMu.Lock()
	defer gamesMu.Unlock()
	if rouletteGames[gameKey(e)] != nil {
		t.Fatal("room retained")
	}
}
