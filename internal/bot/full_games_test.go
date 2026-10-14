package bot

import (
	"encoding/json"
	"qqbot-go/internal/onebot"
	"strings"
	"testing"
)

func TestRouletteTurnAndChamber(t *testing.T) {
	gamesMu.Lock()
	rouletteGames = map[string]*rouletteState{"g:42": {Owner: "1", Players: []string{"1", "2"}, Names: map[string]string{"1": "甲", "2": "乙"}, Bullet: 1, Started: true}}
	gamesMu.Unlock()
	h := &Handler{}
	e := onebot.Event{MessageType: "group", GroupID: json.RawMessage("42"), UserID: json.RawMessage("2")}
	if got := h.fullGameCommand("轮盘赌", "扣扳机", e, "乙"); !strings.Contains(got, "还没轮到你") {
		t.Fatalf("wrong player should be rejected, got %q", got)
	}
	e.UserID = json.RawMessage("1")
	if got := h.fullGameCommand("轮盘赌", "扣扳机", e, "甲"); !strings.Contains(got, "下一位是 乙") {
		t.Fatalf("safe chamber should advance, got %q", got)
	}
}

func TestBlackjackScoringAndRoundAdvance(t *testing.T) {
	if got := handScore([]string{"♠A", "♥K"}); got != 21 {
		t.Fatalf("soft ace score = %d, want 21", got)
	}
	if got := handScore([]string{"♠A", "♥K", "♦9"}); got != 20 {
		t.Fatalf("ace should soften, got %d", got)
	}
	g := &blackjackState{Players: []string{"a", "b"}, Names: map[string]string{"a": "甲", "b": "乙"}, Hands: map[string][]string{"a": {"♠10", "♥8"}, "b": {"♣9", "♦8"}}, Deck: []string{"♠2"}, Dealer: []string{"♠10", "♥6"}, Stood: map[string]bool{"a": true}, Turn: 0, Started: true}
	got := advanceBlackjack(g)
	if !strings.Contains(got, "轮到 乙") {
		t.Fatalf("expected next player, got %q", got)
	}
	g.Stood["b"] = true
	got = advanceBlackjack(g)
	if !strings.Contains(got, "本局结算") || g.Started {
		t.Fatalf("expected settled round, got %q", got)
	}
}

func TestSocialRiddleAndIdiomFlow(t *testing.T) {
	gamesMu.Lock()
	socialGames = map[string]*socialState{}
	gamesMu.Unlock()
	h := &Handler{Prefix: "/"}
	e := onebot.Event{PostType: "message", MessageType: "group", GroupID: json.RawMessage("7"), UserID: json.RawMessage("8")}
	start := h.socialCommand("猜谜", "", e, "玩家")
	if !strings.Contains(start, "猜谜开始") {
		t.Fatalf("riddle did not start: %s", start)
	}
	gamesMu.Lock()
	socialGames["g:7"] = &socialState{Kind: "riddle", Answer: "水", Hint: "越洗越脏"}
	gamesMu.Unlock()
	if got := h.handleSocialMessage(e, "水"); !strings.Contains(got, "回答正确") {
		t.Fatalf("riddle answer not accepted: %q", got)
	}
	gamesMu.Lock()
	socialGames["g:7"] = &socialState{Kind: "idiom", Last: "一心一意"}
	gamesMu.Unlock()
	if got := h.handleSocialMessage(e, "意气风发"); !strings.Contains(got, "接得好") {
		t.Fatalf("valid idiom rejected: %q", got)
	}
}
