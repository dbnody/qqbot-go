package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"qqbot-go/internal/onebot"
	"strings"
	"time"
)

type weatherPayload struct {
	Current []struct {
		Desc []struct {
			Value string `json:"value"`
		} `json:"weatherDesc"`
		Temp     string `json:"temp_C"`
		Feels    string `json:"FeelsLikeC"`
		Humidity string `json:"humidity"`
		Wind     string `json:"windspeedKmph"`
	} `json:"current_condition"`
}

func (h *Handler) basicCommand(name, args, scope, nickname string, e onebot.Event) string {
	switch name {
	case "天气", "weather":
		city := strings.TrimSpace(args)
		if city == "" {
			return "用法：/天气 城市，例如：/天气 上海"
		}
		ctx := http.Client{Timeout: 8 * time.Second}
		resp, err := ctx.Get("https://wttr.in/" + url.PathEscape(city) + "?format=j1")
		if err != nil {
			return "天气服务暂时不可用，请稍后再试或检查城市名称。"
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		var w weatherPayload
		if err != nil || json.Unmarshal(b, &w) != nil || len(w.Current) == 0 || len(w.Current[0].Desc) == 0 {
			return "天气服务暂时不可用，请稍后再试或检查城市名称。"
		}
		c := w.Current[0]
		return fmt.Sprintf("%s 当前天气：%s\n温度：%s℃，体感：%s℃\n湿度：%s%%，风速：%s km/h", city, c.Desc[0].Value, c.Temp, c.Feels, c.Humidity, c.Wind)
	case "今日运势", "运势", "fortune":
		return dailyFortune(scope, nickname)
	case "一言", "今日一句", "quote", "hitokoto":
		quoteScope := "private:" + strings.Trim(string(e.UserID), `"`)
		if e.MessageType == "group" {
			quoteScope = "group:" + strings.Trim(string(e.GroupID), `"`)
		}
		return "今日一句：\n“" + nextQuote(quoteScope) + "”"
	case "塔罗", "塔罗牌", "tarot":
		return "塔罗抽牌需要通过图片命令处理。"
	case "", "你好":
		return ""
	}
	return ""
}
