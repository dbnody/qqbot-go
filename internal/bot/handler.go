package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"qqbot-go/internal/onebot"
	"qqbot-go/internal/store"
	"strings"
)

type Handler struct {
	Client *onebot.Client
	Prefix string
	Store  *store.Store
}

func NewHandler(c *onebot.Client, p string, s *store.Store) *Handler { return &Handler{c, p, s} }
func (h *Handler) HandleEvent(e onebot.Event) {
	if e.PostType == "notice" {
		h.handleNotice(e)
		return
	}
	if e.PostType != "message" {
		return
	}
	if strings.Trim(string(e.UserID), `"`) == strings.Trim(string(e.SelfID), `"`) {
		return
	}
	t := strings.TrimSpace(e.PlainText())
	h.cacheMessage(e)
	memoryScope := "private:" + strings.Trim(string(e.UserID), "\"")
	if e.MessageType == "group" {
		memoryScope = "group:" + strings.Trim(string(e.GroupID), "\"")
	}
	userID := strings.Trim(string(e.UserID), "\"")
	nickname := "QQ用户"
	if sender, ok := e.Raw["sender"].(map[string]any); ok {
		for _, key := range []string{"card", "nickname"} {
			if v, ok := sender[key].(string); ok && v != "" {
				nickname = v
				break
			}
		}
	}
	if e.MessageType == "group" {
		group := strings.Trim(string(e.GroupID), "\"")
		_ = h.Store.RecordMessage(group, userID, nickname, t)
		if reply := h.moderate(e, t, group, userID); reply != "" {
			_ = h.Client.SendMessage(context.Background(), e, reply)
			return
		}
		if response := h.handleBilibili(e, t); response != "" {
			_ = h.Client.SendMessage(context.Background(), e, response)
			return
		}
	}
	if reply := h.handleSocialMessage(e, t); reply != "" {
		_ = h.Client.SendMessage(context.Background(), e, reply)
		return
	}
	if !strings.HasPrefix(t, h.Prefix) {
		if reply := faqForEvent(e, t); reply != "" {
			_ = h.Client.SendMessage(context.Background(), e, reply)
			return
		}
		if reply := h.aiReply(e, memoryScope, t); reply != "" {
			_ = h.Client.SendMessage(context.Background(), e, reply)
		}
		return
	}
	parts := strings.Fields(strings.TrimSpace(strings.TrimPrefix(t, h.Prefix)))
	if len(parts) == 0 {
		return
	}
	name, args := parts[0], strings.Join(parts[1:], " ")
	id := userID
	if e.MessageType == "group" {
		id = "group:" + strings.Trim(string(e.GroupID), "\"") + ":" + id
	}
	reply := ""
	switch name {
	case "ping", "测试":
		reply = "pong！QQ Bot Go 正常运行。"
	case "help", "帮助":
		reply = helpText(h.Prefix)
	case "签到", "sign", "打卡":
		ok, r, total, err := h.Store.SignIn(id, nickname)
		if err != nil {
			log.Printf("sign-in: %v", err)
			reply = "签到失败，请稍后再试。"
		} else if !ok {
			reply = "今天已经签到过啦，明天再来吧。"
		} else {
			reply = fmt.Sprintf("签到成功！获得 %d 点灵气，累计签到 %d 天。", r, total)
		}
	case "修仙", "修炼":
		reply = h.cultivation(id, nickname, args)
	case "天气", "weather", "今日运势", "运势", "fortune", "一言", "今日一句", "quote", "hitokoto", "塔罗", "塔罗牌", "tarot":
		scope := id
		if e.MessageType == "group" {
			scope = "group:" + strings.Trim(string(e.GroupID), "\"") + ":" + userID
		}
		if name == "塔罗" || name == "塔罗牌" || name == "tarot" {
			reply = h.tarotCommand(e, args)
		} else {
			reply = h.basicCommand(name, args, scope, nickname, e)
		}
	case "群统计", "消息统计", "统计", "发言排行", "排行榜", "发言榜":
		reply = h.communityCommand(name, args, e)
	case "词云", "群词云", "wordcloud":
		if e.MessageType != "group" {
			reply = "词云功能只能在群聊中使用。"
		} else {
			days := parseDays(args, 7)
			path, err := h.wordCloud(strings.Trim(string(e.GroupID), `"`), days)
			if path != "" {
				defer os.Remove(path)
			}
			if err != nil {
				log.Printf("word cloud: %v", err)
				reply = "词云生成失败，请稍后再试。"
			} else if path == "" {
				reply = "最近没有足够的群聊文本来生成词云。"
			} else if err = h.sendImage(e, path, ""); err != nil {
				log.Printf("send word cloud: %v", err)
				reply = "词云已生成，但图片发送失败，请检查 NapCat 图片路径配置。"
			}
		}
	case "meme模板", "梗图模板", "模板列表":
		reply = memeTemplateList()
	case "meme", "梗图", "表情包", "做图":
		if strings.TrimSpace(args) == "" || args == "模板" || args == "列表" {
			reply = memeTemplateList()
		} else {
			path, err := h.makeMeme(args)
			if path != "" {
				defer os.Remove(path)
			}
			if err != nil {
				log.Printf("meme: %v", err)
				reply = "Meme 生成失败：模板不存在或图片渲染失败。发送 /meme模板 查看可用模板。"
			} else if err = h.sendImage(e, path, ""); err != nil {
				log.Printf("send meme: %v", err)
				reply = "Meme 已生成，但图片发送失败，请检查 NapCat 图片路径配置。"
			}
		}
	case "群管", "群管理", "群管帮助", "ban", "禁言", "mute", "unban", "解禁", "解除禁言", "unmute", "ta", "踢人", "踢出", "kick":
		reply = h.groupAdmin(name, args, e)
	case "清除记忆", "清空记忆", "忘记本群":
		if e.MessageType == "group" {
			if !h.isBotAdmin(userID) {
				reply = "群聊中只有 Bot 管理员可以清除本群的 AI 记忆。"
			} else if h.Store.ClearMemory(memoryScope) == nil {
				reply = "已清除本群的 AI 记忆，不会影响其他群。"
			}
		} else if h.Store.ClearMemory(memoryScope) == nil {
			reply = "已清除你的私聊 AI 记忆，不会影响其他用户。"
		}
	case "投胎", "reborn", "转世", "投胎模拟器":
		reply = rebornStory(nickname)
	case "cp", "CP", "群内配对", "CP抽签", "群内CP":
		reply = h.pairing(e)
	case "点歌", "点歌台", "music":
		reply = h.musicCommand(e, args)
	case "锐评", "评价", "吐槽":
		reply = h.reviewCommand(e, args, nickname, false)
	case "锐评头像", "头像锐评":
		reply = h.reviewCommand(e, args, nickname, true)
	case "21点", "21", "二十一点", "blackjack", "轮盘赌", "俄罗斯轮盘", "左轮赌", "roulette":
		reply = h.fullGameCommand(name, args, e, nickname)
	case "猜谜", "猜谜语", "谜语", "成语接龙", "接龙", "idiom", "你画我猜", "画猜", "draw", "猜歌名", "猜歌", "听歌猜名", "song":
		reply = h.socialCommand(name, args, e, nickname)
	default:
		reply = h.basicCommand(name, args, id, nickname, e)
	}
	if reply != "" {
		_ = h.Client.SendMessage(context.Background(), e, reply)
	}
}

func (h *Handler) cultivation(id, n, a string) string {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "修炼", "闭关", "练功", "cultivate":
		p, g, e := h.Store.Cultivate(id, n)
		if e != nil {
			return "修炼失败，请稍后再试。"
		}
		th := fmt.Sprint(p.Threshold)
		if p.Threshold == 0 {
			th = "圆满"
		}
		return fmt.Sprintf("你修炼了一轮，获得 %d 点修为。\n当前境界：%s，修为：%d/%s，灵石：%d", g, p.Name, p.Experience, th, p.Stones)
	case "突破", "breakthrough":
		old, next, ok, max, p, e := h.Store.Breakthrough(id, n)
		if e != nil {
			return "突破失败，请稍后再试。"
		}
		if max {
			return "你已达到渡劫期，暂时没有更高境界。"
		}
		if !ok {
			return fmt.Sprintf("突破失败，还需要修炼至 %d 点修为。当前：%d/%d。", p.Threshold, p.Experience, p.Threshold)
		}
		return fmt.Sprintf("突破成功！恭喜从 %s 晋升为 %s！", old, next)
	default:
		p, e := h.Store.Profile(id, n)
		if e != nil {
			return "查看修仙档案失败，请稍后再试。"
		}
		th := fmt.Sprint(p.Threshold)
		if p.Threshold == 0 {
			th = "圆满"
		}
		return fmt.Sprintf("修士：%s\n境界：%s\n修为：%d/%s\n灵石：%d\n\n用法：/修仙 修炼 或 /修仙 突破", n, p.Name, p.Experience, th, p.Stones)
	}
}
