package bot

import (
	"context"
	"fmt"
	"qqbot-go/internal/onebot"
	"strings"
)

func (h *Handler) communityCommand(name, args string, e onebot.Event) string {
	if e.MessageType != "group" {
		return "群统计功能只能在群聊中使用。"
	}
	days := parseDays(args, 1)
	group := strings.Trim(string(e.GroupID), "\"")
	if name == "发言排行" || name == "排行榜" || name == "发言榜" {
		days = parseDays(args, 7)
		out, err := h.Store.GroupRanking(group, days)
		if err != nil {
			return "统计暂时不可用。"
		}
		return out
	}
	messages, users, err := h.Store.GroupStats(group, days)
	if err != nil {
		return "统计暂时不可用。"
	}
	return fmt.Sprintf("最近 %d 天群统计：\n消息：%d 条\n活跃成员：%d 人", days, messages, users)
}
func (h *Handler) handleNotice(e onebot.Event) {
	if e.PostType != "notice" {
		return
	}
	if e.Raw["notice_type"] == "group_recall" || e.Raw["notice_type"] == "friend_recall" {
		h.handleRecall(e)
		return
	}
	if e.Raw["notice_type"] == "notify" && e.Raw["sub_type"] == "poke" {
		h.poke(e)
		return
	}
	notice, _ := e.Raw["notice_type"].(string)
	sub, _ := e.Raw["sub_type"].(string)
	if notice != "group_increase" && notice != "group_decrease" {
		return
	}
	message := ""
	user := strings.Trim(string(e.UserID), "\"")
	if notice == "group_increase" {
		message = "欢迎新成员 " + user + " 加入本群！发送 /help 可以查看 Bot 功能。"
	} else {
		verb := "离开"
		if sub == "kick" {
			verb = "被移出"
		}
		message = "成员 " + user + " 已" + verb + "本群，祝一路顺风。"
	}
	_, _ = h.Client.Call(context.Background(), "send_group_msg", map[string]any{"group_id": e.GroupID, "message": message})
}
