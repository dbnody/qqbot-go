package bot

import (
	"qqbot-go/internal/onebot"
	"regexp"
	"strconv"
	"strings"
)

func helpText(prefix string) string {
	return strings.ReplaceAll("可用命令（方括号为可选参数）：\n/ping（测试）、/help（帮助）\n/天气 城市、/今日运势（运势）、/一言（今日一句）\n/塔罗 [问题]、/投胎、/cp（群内配对）\n/签到（打卡）、/修仙 [修炼|突破]\n/锐评 内容/图片/@成员、/锐评头像 @成员\n私聊可直接聊天；群聊 @Bot 可聊天或询问图片。\n/清除记忆（群聊限配置的 Bot 管理员）\n/点歌 [qq|网易云] 歌手 歌名\n/meme 模板 文字（梗图/表情包/做图）、/meme模板\n/群统计 [天数]、/发言排行 [天数]、/词云 [天数]\n/猜谜 [提示|结束]、/成语接龙 [结束]\n/你画我猜 [结束]、/猜歌名 [提示|结束]\n/轮盘赌 单人|多人|加入|开始|扣扳机|状态|退出\n/21点 创建|加入|开始|要牌|停牌|状态|退出\n/群管帮助、/ban @成员 时长、/unban @成员、/ta @成员\nB站链接自动解析；戳一戳、入退群通知、违禁词处理和撤回监控依配置启用。", "/", prefix)
}

var dayNumberRE = regexp.MustCompile(`\d+`)

func parseDays(args string, fallback int) int {
	if s := dayNumberRE.FindString(args); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n > 365 {
			return 365
		}
		if n < 1 {
			return 1
		}
		return n
	}
	return fallback
}

var introRE = regexp.MustCompile(`^(?:你是谁|你叫什么|介绍一下自己)[？?！!。.]?$`)
var greetingRE = regexp.MustCompile(`^(?:你好|您好|嗨|hi|hello)[！!。,. ]*$`)
var thanksRE = regexp.MustCompile(`^(?:谢谢|感谢|多谢)[！!。.]?$`)
var capabilitiesRE = regexp.MustCompile(`^(?:你能做什么|你会什么|有什么功能)[？?！!。.]?$`)
var modelRE = regexp.MustCompile(`你是(?:什么|哪种)模型`)

func faqForEvent(e onebot.Event, t string) string {
	t = strings.TrimSpace(t)
	switch {
	case modelRE.MatchString(t):
		if e.MessageType == "group" && !mentioned(e) {
			return ""
		}
		return "呐，我是小肥，由我的主人创造，现在正以小机器人的身份陪在你身边喵。\n如果你有什么想知道的，或者需要帮忙的事情，随时告诉我哦！"
	case introRE.MatchString(t):
		return "我是小肥，一个QQ Bot，可以陪你聊天和玩小游戏。"
	case greetingRE.MatchString(t):
		return "你好！发送 /help 可以查看我现在支持的功能。"
	case thanksRE.MatchString(t):
		return "不客气，随时可以叫我。"
	case capabilitiesRE.MatchString(t):
		return "我可以进行天气的查询和一些简单的小游戏。发送 /help 查看用法。"
	}
	return ""
}
