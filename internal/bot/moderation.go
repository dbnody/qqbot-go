package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"qqbot-go/internal/onebot"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var atRE = regexp.MustCompile("@?([0-9]{5,})")
var durRE = regexp.MustCompile("(?i)([0-9]+)\\s*(分钟|分|min|m|小时|时|hour|h|天|日|day|d)?")

func (h *Handler) moderate(e onebot.Event, text, group, user string) string {
	if os.Getenv("QQBOT_FORBIDDEN_WORDS") == "" {
		return ""
	}
	role := "member"
	if s, ok := e.Raw["sender"].(map[string]any); ok {
		role, _ = s["role"].(string)
	}
	if role == "admin" || role == "owner" {
		return ""
	}
	found := ""
	for _, w := range strings.Split(os.Getenv("QQBOT_FORBIDDEN_WORDS"), ",") {
		w = strings.TrimSpace(w)
		if w != "" && strings.Contains(strings.ToLower(text), strings.ToLower(w)) {
			found = w
			break
		}
	}
	if found == "" {
		return ""
	}
	var self any = e.SelfID
	resp, err := h.Client.Call(context.Background(), "get_group_member_info", map[string]any{"group_id": e.GroupID, "user_id": self})
	if err != nil {
		return ""
	}
	var info struct {
		Data struct {
			Role string `json:"role"`
		} `json:"data"`
	}
	_ = json.Unmarshal(resp, &info)
	if info.Data.Role != "owner" && info.Data.Role != "admin" {
		return ""
	}
	if _, err := h.Client.Call(context.Background(), "delete_msg", map[string]any{"message_id": e.MessageID}); err != nil {
		return ""
	}
	seconds, err := strconv.Atoi(os.Getenv("QQBOT_FORBIDDEN_MUTE_SECONDS"))
	if err != nil {
		seconds = 60
	}
	if seconds > 2592000 {
		seconds = 2592000
	}
	if seconds > 0 {
		if _, err := h.Client.Call(context.Background(), "set_group_ban", map[string]any{"group_id": e.GroupID, "user_id": e.UserID, "duration": seconds}); err != nil {
			return "检测到违禁词，已删除消息；禁言操作失败。"
		}
	}
	return fmt.Sprintf("检测到违禁词，已删除相关消息并处理（关键词：%s）。", found)
}
func (h *Handler) groupAdmin(cmd, args string, e onebot.Event) string {
	if e.MessageType != "group" {
		return "群管理命令只能在群聊中使用。"
	}
	if cmd == "群管" || cmd == "群管理" || cmd == "群管帮助" {
		return "群管理命令：\n/ban @成员 时长 - 禁言（最多30天）\n/unban @成员 - 解除禁言\n/ta @成员 - 移出成员"
	}
	role := "member"
	if s, ok := e.Raw["sender"].(map[string]any); ok {
		role, _ = s["role"].(string)
	}
	if role != "owner" && role != "admin" {
		return "只有群主和群管理员可以使用群管理功能。"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var self any = e.SelfID
	raw, err := h.Client.Call(ctx, "get_group_member_info", map[string]any{"group_id": e.GroupID, "user_id": self})
	if err != nil {
		return "我目前不是本群管理员，无法执行群管理操作。"
	}
	var inf struct {
		Data struct {
			Role string `json:"role"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &inf)
	if inf.Data.Role != "owner" && inf.Data.Role != "admin" {
		return "我目前不是本群管理员，无法执行群管理操作。"
	}
	target := mentionedTarget(e)
	if target == "" {
		match := atRE.FindStringSubmatch(args)
		if len(match) >= 2 {
			target = match[1]
		}
	}
	if target == "" {
		return "用法：/" + cmd + " @成员"
	}
	if target == strings.Trim(string(e.SelfID), `"`) {
		return "不能对我自己执行这项操作。"
	}
	var targetID any = json.Number(target)
	raw, err = h.Client.Call(ctx, "get_group_member_info", map[string]any{"group_id": e.GroupID, "user_id": targetID})
	if err != nil {
		return "找不到这个群成员，请确认 QQ 号或 @对象正确。"
	}
	var tinf struct {
		Data struct {
			Role string `json:"role"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &tinf)
	if tinf.Data.Role == "owner" || tinf.Data.Role == "admin" {
		return "不能对群主或其他管理员执行这项操作。"
	}
	action := "set_group_ban"
	params := map[string]any{"group_id": e.GroupID, "user_id": targetID}
	switch cmd {
	case "ban", "禁言", "mute":
		mins := 1
		durationArgs := args
		if mentionedTarget(e) == "" {
			durationArgs = strings.Replace(args, target, "", 1)
		}
		if d := durRE.FindStringSubmatch(durationArgs); len(d) > 1 {
			n, _ := strconv.Atoi(d[1])
			mins = n
			if len(d) > 2 {
				u := strings.ToLower(d[2])
				if strings.Contains(u, "时") || u == "h" || u == "hour" {
					mins = n * 60
				} else if strings.Contains(u, "天") || u == "日" || u == "d" || u == "day" {
					mins = n * 1440
				}
			}
		}
		if mins > 43200 {
			mins = 43200
		}
		if mins < 1 {
			mins = 1
		}
		params["duration"] = mins * 60
	case "unban", "解禁", "解除禁言", "unmute":
		params["duration"] = 0
	case "ta", "踢人", "踢出", "kick":
		action = "set_group_kick"
	default:
		return "未知命令。"
	}
	if _, err = h.Client.Call(ctx, action, params); err != nil {
		return "操作失败，请确认机器人权限和目标仍在群内。"
	}
	if action == "set_group_kick" {
		return "已将 " + target + " 移出本群。"
	}
	secs, _ := params["duration"].(int)
	if secs == 0 {
		return "已解除 " + target + " 的禁言。"
	}
	return fmt.Sprintf("已将 %s 禁言 %d 分钟。", target, secs/60)
}

func mentionedTarget(e onebot.Event) string {
	if len(e.Message) > 0 {
		target, _ := reviewSources(e)
		return target
	}
	raw, ok := e.Raw["message"]
	if !ok {
		return ""
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return ""
	}
	var segments []struct {
		Type string         `json:"type"`
		Data map[string]any `json:"data"`
	}
	if json.Unmarshal(b, &segments) != nil {
		return ""
	}
	for _, seg := range segments {
		if seg.Type != "at" {
			continue
		}
		switch v := seg.Data["qq"].(type) {
		case string:
			if v != "all" {
				return v
			}
		case json.Number:
			return v.String()
		case float64:
			return strconv.FormatInt(int64(v), 10)
		}
	}
	return ""
}
