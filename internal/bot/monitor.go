package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"qqbot-go/internal/onebot"
	"strings"
	"sync"
	"time"
)

var recentMessages = struct {
	sync.Mutex
	items map[string]json.RawMessage
	order []string
}{items: make(map[string]json.RawMessage)}

func cacheRecallMessage(key string, message json.RawMessage) {
	recentMessages.Lock()
	defer recentMessages.Unlock()
	if _, ok := recentMessages.items[key]; !ok {
		recentMessages.order = append(recentMessages.order, key)
	}
	recentMessages.items[key] = append(json.RawMessage(nil), message...)
	for len(recentMessages.order) > 2000 {
		delete(recentMessages.items, recentMessages.order[0])
		recentMessages.order = recentMessages.order[1:]
	}
}

func recallSegments(raw json.RawMessage) []map[string]any {
	parsed := onebot.ParseMessage(raw)
	var segments []map[string]any
	for _, seg := range parsed {
		segments = append(segments, map[string]any{"type": seg.Type, "data": seg.Data})
	}
	for _, seg := range segments {
		switch seg["type"] {
		case "image", "video", "record", "file":
			if data, ok := seg["data"].(map[string]any); ok {
				if u, ok := data["url"].(string); ok && u != "" {
					data["file"] = u
				}
			}
		}
	}
	return segments
}

func messageKey(group string, id json.RawMessage) string {
	return group + ":" + strings.Trim(string(id), "\"")
}
func (h *Handler) cacheMessage(e onebot.Event) {
	if strings.Trim(string(e.UserID), `"`) == strings.Trim(string(e.SelfID), `"`) || len(e.MessageID) == 0 {
		return
	}
	group := strings.Trim(string(e.GroupID), "\"")
	cacheRecallMessage(messageKey(group, e.MessageID), e.Message)
}
func (h *Handler) handleRecall(e onebot.Event) {
	admins := configuredAdmins()
	if len(admins) == 0 {
		return
	}
	group := strings.Trim(string(e.GroupID), "\"")
	user := strings.Trim(string(e.UserID), "\"")
	self := strings.Trim(string(e.SelfID), `"`)
	if user == self || fmt.Sprint(e.Raw["operator_id"]) == self {
		return
	}
	id, _ := json.Marshal(e.Raw["message_id"])
	key := messageKey(group, id)
	recentMessages.Lock()
	content, ok := recentMessages.items[key]
	recentMessages.Unlock()
	if !ok {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		raw, err := h.Client.Call(ctx, "get_msg", map[string]any{"message_id": id})
		cancel()
		var result struct {
			Data struct {
				Message json.RawMessage `json:"message"`
			} `json:"data"`
		}
		if err != nil || json.Unmarshal(raw, &result) != nil || len(result.Data.Message) == 0 {
			return
		}
		content = result.Data.Message
	}
	source := "私聊，发送者 QQ " + user
	if group != "" {
		source = fmt.Sprintf("群聊 %s，发送者 QQ %s，操作者 QQ %v", group, user, e.Raw["operator_id"])
	}
	message := []map[string]any{{"type": "text", "data": map[string]any{"text": fmt.Sprintf("[撤回消息]\n来源：%s\n消息 ID：%s\n", source, strings.Trim(string(id), `"`))}}}
	message = append(message, recallSegments(content)...)
	allOK := true
	for _, admin := range admins {
		admin = strings.TrimSpace(admin)
		if admin == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		_, err := h.Client.Call(ctx, "send_private_msg", map[string]any{"user_id": admin, "message": message})
		cancel()
		if err != nil {
			allOK = false
		}
	}
	if allOK {
		recentMessages.Lock()
		delete(recentMessages.items, key)
		for i, k := range recentMessages.order {
			if k == key {
				recentMessages.order = append(recentMessages.order[:i], recentMessages.order[i+1:]...)
				break
			}
		}
		recentMessages.Unlock()
	}
}
func (h *Handler) poke(e onebot.Event) {
	target := strings.Trim(fmt.Sprint(e.Raw["target_id"]), "\"")
	self := strings.Trim(string(e.SelfID), "\"")
	user := strings.Trim(string(e.UserID), "\"")
	if target != self || user == self {
		return
	}
	params := map[string]any{"user_id": e.UserID}
	if len(e.GroupID) > 0 {
		params["group_id"] = e.GroupID
	}
	if secureIndex(2) == 0 {
		if _, err := h.Client.Call(context.Background(), "send_poke", params); err == nil {
			return
		}
	}
	texts := []string{"喵？谁在戳我！", "再戳就要反击了喵。", "检测到一只调皮的群友。", "别戳啦，认真说话喵。", "戳戳收到，给你一记可爱的回礼！", "喵呜——被你发现啦！", "轻一点喵，我可是会害羞的。", "你戳一下，我记你一次喵。", "警告：检测到可疑戳戳行为！", "今天也要和 Bot 好好相处喵。", "收到你的戳戳，心情增加一点点。", "再戳一下试试？我可要反戳啦！", "这是在召唤我吗，喵？", "戳戳能量已收集完毕！", "摸鱼被抓到了，先回你一个戳。", "喵喵喵？有事请直接说喵。", "你的戳戳申请已通过。", "别闹啦，给你一个友好的猫爪印。"}
	msg := texts[secureIndex(len(texts))]
	if len(e.GroupID) > 0 {
		_, _ = h.Client.Call(context.Background(), "send_group_msg", map[string]any{"group_id": e.GroupID, "message": msg})
	} else {
		_, _ = h.Client.Call(context.Background(), "send_private_msg", map[string]any{"user_id": e.UserID, "message": msg})
	}
}
