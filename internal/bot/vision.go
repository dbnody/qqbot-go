package bot

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"qqbot-go/internal/onebot"
)

func reviewSources(e onebot.Event) (target, source string) {
	for _, s := range onebot.ParseMessage(e.Message) {
		value := func(key string) string {
			if s.Data[key] == nil {
				return ""
			}
			return fmt.Sprint(s.Data[key])
		}
		if (s.Type == "at" || s.Type == "mention") && target == "" {
			v := value("qq")
			if v == "" {
				v = value("user_id")
			}
			if _, err := strconv.ParseUint(v, 10, 64); err == nil {
				target = v
			}
		}
		if s.Type == "image" && source == "" {
			source = value("url")
			if source == "" {
				source = value("file")
			}
		}
	}
	return
}
func (h *Handler) resolveImageSource(ctx context.Context, source string) (string, error) {
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		return source, nil
	}
	if h.Client == nil {
		return "", fmt.Errorf("not connected")
	}
	lookup, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	raw, err := h.Client.Call(lookup, "get_image", map[string]any{"file": source})
	if err != nil {
		return "", err
	}
	var result struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &result) != nil || result.Data.URL == "" {
		return "", fmt.Errorf("image URL unavailable")
	}
	return result.Data.URL, nil
}
func imageDataURL(ctx context.Context, source string) (string, error) {
	u, err := url.Parse(source)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return "", fmt.Errorf("image requires HTTP URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image HTTP %d", resp.StatusCode)
	}
	const limit = 8 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return "", err
	}
	if len(data) > limit {
		return "", fmt.Errorf("image too large")
	}
	mime := http.DetectContentType(data)
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
	default:
		return "", fmt.Errorf("unsupported image format")
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func visualReview(ctx context.Context, source, question string) string {
	data, err := imageDataURL(ctx, source)
	if err != nil {
		return "图片读取失败，请重新发送有效图片（最大 8 MiB）。"
	}
	answer, err := askAI(ctx, []aiMessage{{"system", aiSystemPrompt}, {"user", []any{map[string]any{"type": "text", "text": "请友善幽默、有分寸地评价图片的构图、色彩、光线和氛围，120字以内。\n" + question}, map[string]any{"type": "image_url", "image_url": map[string]string{"url": data}}}}})
	if err != nil {
		switch err {
		case aiHTTPError(400), aiHTTPError(404), aiHTTPError(415):
			return "当前 AI 模型或接口可能不支持图片理解，请检查视觉模型配置。"
		}
		return "图片 AI 服务暂时不可用，请检查模型和接口配置。"
	}
	return answer
}
func (h *Handler) reviewCommand(e onebot.Event, args, nickname string, avatar bool) string {
	if os.Getenv("AI_API_KEY") == "" {
		return "AI 接口尚未配置，暂时不能进行锐评。"
	}
	target, source := reviewSources(e)
	if avatar {
		source = ""
		if target == "" && e.MessageType == "private" {
			target = strings.Trim(string(e.UserID), `"`)
		}
		if target == "" {
			return "用法：/锐评头像 @成员；私聊中可直接锐评自己的头像。"
		}
		if target == strings.Trim(string(e.SelfID), `"`) {
			return "不评价我自己的头像啦，换一位成员试试喵。"
		}
	}
	if source == "" && target != "" {
		source = "https://q1.qlogo.cn/g?b=qq&nk=" + url.QueryEscape(target) + "&s=640"
	}
	if source == "" {
		if strings.TrimSpace(args) == "" {
			return "用法：/锐评 内容、/锐评 图片，或 /锐评 @成员。"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		cleanup := h.processingNotice(ctx, e, "正在准备锐评，请稍候……")
		defer cleanup()
		return aiTextReview(args, nickname)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
	var err error
	source, err = h.resolveImageSource(ctx, source)
	if err != nil {
		return "无法取得图片下载地址，请重新发送图片，或检查 NapCat get_image 返回值。"
	}
	display := nickname
	if target != "" {
		display = target
		if e.MessageType == "group" {
			lookup, c := context.WithTimeout(ctx, 5*time.Second)
			raw, err := h.Client.Call(lookup, "get_group_member_info", map[string]any{"group_id": e.GroupID, "user_id": json.Number(target)})
			c()
			if err == nil {
				var r struct {
					Data struct {
						Card     string `json:"card"`
						Nickname string `json:"nickname"`
					} `json:"data"`
				}
				if json.Unmarshal(raw, &r) == nil {
					if r.Data.Card != "" {
						display = r.Data.Card
					} else if r.Data.Nickname != "" {
						display = r.Data.Nickname
					}
				}
			}
		}
	}
	cleanup := h.processingNotice(ctx, e, "正在读取图片并进行锐评，请稍候……")
	defer cleanup()
	question := strings.TrimSpace(args)
	if question == "" {
		question = "请对这张图片做一段友善、幽默的审美锐评。"
	}
	return display + " 的锐评：\n" + visualReview(ctx, source, question)
}
