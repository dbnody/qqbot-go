package bot

import (
	"context"
	"qqbot-go/internal/onebot"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func parseMusicRequest(args string) (string, string) {
	q := strings.TrimSpace(args)
	platform := "both"
	re := regexp.MustCompile("(?i)^(qq音乐|qq|腾讯音乐|网易云音乐|网易云|网易|163|netease)\\s+(.+)$")
	if m := re.FindStringSubmatch(q); len(m) > 2 {
		if strings.Contains(strings.ToLower(m[1]), "网易") || m[1] == "163" || strings.EqualFold(m[1], "netease") {
			platform = "163"
		} else {
			platform = "qq"
		}
		q = m[2]
	}
	labeled := regexp.MustCompile(`^(?:歌手名?|歌手)\s*[:：]\s*(.+?)\s+(?:歌曲?|歌名)\s*[:：]\s*(.+)$`)
	if m := labeled.FindStringSubmatch(q); len(m) == 3 {
		q = m[1] + " " + m[2]
	} else {
		q = regexp.MustCompile(`^(?:歌曲?|歌名|歌手名?)\s*[:：]\s*`).ReplaceAllString(q, "")
		q = regexp.MustCompile(`\s*[-—|]\s*`).ReplaceAllString(q, " ")
	}
	q = strings.Join(strings.Fields(q), " ")
	if len([]rune(q)) > 100 {
		q = string([]rune(q)[:100])
	}
	return q, platform
}

func musicSegments(text string, songs []songResult) []map[string]any {
	message := []map[string]any{{"type": "text", "data": map[string]any{"text": text}}}
	for _, song := range songs {
		message = append(message, map[string]any{"type": "music", "data": map[string]any{"type": song.Platform, "id": strconv.Itoa(song.ID)}})
	}
	return message
}

func (h *Handler) musicCommand(e onebot.Event, args string) string {
	q, platform := parseMusicRequest(args)
	if q == "" {
		return "用法：/点歌 歌名或歌手 歌名；可用 qq 或 网易云 指定平台。"
	}
	text, songs := searchMusicResults(q, platform)
	if len(songs) == 0 {
		return text
	}
	params := map[string]any{"message": musicSegments(text, songs)}
	if e.MessageType == "group" {
		params["group_id"] = e.GroupID
	} else {
		params["user_id"] = e.UserID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := h.Client.Call(ctx, "send_msg", params); err != nil {
		return text + "\n音乐卡片发送失败，以上链接仍可打开。"
	}
	return ""
}
