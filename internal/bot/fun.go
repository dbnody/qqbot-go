package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"qqbot-go/internal/onebot"
	"strings"
	"sync"
	"time"
)

var pairingMu sync.Mutex

func (h *Handler) pairing(e onebot.Event) string {
	if e.MessageType != "group" {
		return "CP 抽签是群内玩法，请在群聊中发送 /cp。"
	}
	group := strings.Trim(string(e.GroupID), `"`)
	today := time.Now().Format("2006-01-02")
	pairingMu.Lock()
	defer pairingMu.Unlock()
	first, second, err := h.Store.DailyPair(group, today)
	if err != nil {
		return "CP 抽签记录读取失败，请稍后再试。"
	}
	if first != "" {
		return fmt.Sprintf("本群今天已经 CP 抽签过啦：%s × %s\n明天再来抽新的配对吧！", first, second)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	raw, err := h.Client.Call(ctx, "get_group_member_list", map[string]any{"group_id": e.GroupID})
	if err != nil {
		return "暂时无法获取群成员列表，CP 抽签失败。"
	}
	var result struct {
		Data []map[string]any `json:"data"`
	}
	if json.Unmarshal(raw, &result) != nil {
		return "群内可配对成员不足两人，暂时无法抽签。"
	}
	var candidates []map[string]any
	seen := map[string]bool{}
	for _, m := range result.Data {
		uid := fmt.Sprint(m["user_id"])
		if uid != "<nil>" && uid != strings.Trim(string(e.SelfID), `"`) && m["unfriendly"] != true && !seen[uid] {
			seen[uid] = true
			candidates = append(candidates, m)
		}
	}
	if len(candidates) < 2 {
		return "群内可配对成员不足两人，暂时无法抽签。"
	}
	a, b := candidates[rand.Intn(len(candidates))], candidates[rand.Intn(len(candidates))]
	for fmt.Sprint(a["user_id"]) == fmt.Sprint(b["user_id"]) {
		b = candidates[rand.Intn(len(candidates))]
	}
	name := func(m map[string]any) string {
		for _, k := range []string{"card", "nickname", "user_id"} {
			if v := strings.TrimSpace(fmt.Sprint(m[k])); v != "" && v != "<nil>" {
				return v
			}
		}
		return "群友"
	}
	first, second = name(a), name(b)
	saved, err := h.Store.SaveDailyPair(group, today, first, second)
	if err != nil {
		return "CP 抽签记录保存失败，请稍后再试。"
	}
	if !saved {
		first, second, err = h.Store.DailyPair(group, today)
		if err != nil {
			return "CP 抽签记录读取失败。"
		}
		return fmt.Sprintf("本群今天已经 CP 抽签过啦：%s × %s", first, second)
	}
	stories := catalog.Stories
	return fmt.Sprintf("【今日 CP 抽签】\n%s × %s\n\n%s\n\n今日配对已锁定，明天再来抽新的！", name(a), name(b), stories[rand.Intn(len(stories))])
}

var _ onebot.Event
