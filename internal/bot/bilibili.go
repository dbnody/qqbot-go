package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"qqbot-go/internal/onebot"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type biliVideo struct {
	BVID     string
	CID      int64
	Title    string
	Duration int
	Size     int64
	URL      string
}

var bvRE = regexp.MustCompile(`(?i)\b(BV[0-9A-Za-z]{10})\b`)
var avRE = regexp.MustCompile(`(?i)\bav(\d+)\b`)
var b23RE = regexp.MustCompile(`https?://b23\.tv/[A-Za-z0-9]+`)
var biliBusy sync.Map

func videoLimits() (int64, int) {
	mb, e := strconv.Atoi(os.Getenv("BILIBILI_MAX_MB"))
	if e != nil {
		mb = 50
	}
	if mb < 5 {
		mb = 5
	}
	if mb > 200 {
		mb = 200
	}
	sec, e := strconv.Atoi(os.Getenv("BILIBILI_MAX_SECONDS"))
	if e != nil {
		sec = 600
	}
	if sec < 30 {
		sec = 30
	}
	if sec > 3600 {
		sec = 3600
	}
	return int64(mb) << 20, sec
}
func (h *Handler) handleBilibili(e onebot.Event, text string) string {
	if e.MessageType != "group" {
		return ""
	}
	source := text + " " + string(e.Message)
	source = strings.ReplaceAll(source, `\/`, "/")
	for i := 0; i < 2; i++ {
		if decoded, err := url.PathUnescape(source); err == nil {
			source = decoded
		}
	}
	if !bvRE.MatchString(source) && !avRE.MatchString(source) && !b23RE.MatchString(source) {
		return ""
	}
	group := strings.Trim(string(e.GroupID), `"`)
	if _, loaded := biliBusy.LoadOrStore(group, true); loaded {
		return ""
	}
	defer biliBusy.Delete(group)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cleanup := h.processingNotice(ctx, e, "正在解析 B 站视频，请稍候……")
	defer cleanup()
	video, err := resolveBili(ctx, source)
	if err != nil {
		return "B站视频处理失败：" + err.Error()
	}
	path, err := downloadBili(ctx, video)
	if err != nil {
		return "B站视频处理失败：" + err.Error()
	}
	defer os.Remove(path)
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "B站视频文件路径解析失败。"
	}
	fileURL := (&url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(filepath.ToSlash(absolute), "/")}).String()
	message := []any{map[string]any{"type": "text", "data": map[string]any{"text": "B站视频：" + video.Title + "\n"}}, map[string]any{"type": "video", "data": map[string]any{"file": fileURL}}}
	_, err = h.Client.Call(ctx, "send_group_msg", map[string]any{"group_id": e.GroupID, "message": message})
	if err != nil {
		return "B站视频已下载，但 NapCat 上传失败，请检查视频发送支持。"
	}
	_ = group
	return ""
}
func resolveBili(ctx context.Context, text string) (biliVideo, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	kind, value := "", ""
	if m := bvRE.FindStringSubmatch(text); len(m) > 1 {
		kind = "bvid"
		value = "BV" + m[1][2:]
	} else if m := avRE.FindStringSubmatch(text); len(m) > 1 {
		kind = "aid"
		value = m[1]
	} else if short := b23RE.FindString(text); short != "" {
		req, _ := http.NewRequestWithContext(ctx, "GET", short, nil)
		resp, e := client.Do(req)
		if e != nil {
			return biliVideo{}, e
		}
		resp.Body.Close()
		loc := resp.Request.URL.String()
		if m := bvRE.FindStringSubmatch(loc); len(m) > 1 {
			kind = "bvid"
			value = "BV" + m[1][2:]
		} else if m := avRE.FindStringSubmatch(loc); len(m) > 1 {
			kind = "aid"
			value = m[1]
		}
	}
	if kind == "" {
		return biliVideo{}, errors.New("无法识别视频链接")
	}
	viewURL := "https://api.bilibili.com/x/web-interface/view?" + url.Values{kind: {value}}.Encode()
	req, _ := http.NewRequestWithContext(ctx, "GET", viewURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	resp, e := client.Do(req)
	if e != nil {
		return biliVideo{}, e
	}
	defer resp.Body.Close()
	var view struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			BVID     string `json:"bvid"`
			Title    string `json:"title"`
			Duration int    `json:"duration"`
			Pages    []struct {
				CID int64 `json:"cid"`
			} `json:"pages"`
		} `json:"data"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&view) != nil || view.Code != 0 || len(view.Data.Pages) == 0 {
		return biliVideo{}, errors.New("视频不存在或不可访问")
	}
	page := 1
	if m := regexp.MustCompile(`[?&]p=(\d+)`).FindStringSubmatch(text); len(m) > 1 {
		n, _ := strconv.Atoi(m[1])
		page = max(1, min(n, len(view.Data.Pages)))
	}
	cid := view.Data.Pages[page-1].CID
	params := url.Values{"cid": {strconv.FormatInt(cid, 10)}, "qn": {"16"}, "fnval": {"0"}, "fnver": {"0"}, "fourk": {"0"}}
	if kind == "bvid" {
		params.Set("bvid", value)
	} else {
		params.Set("avid", value)
	}
	req, _ = http.NewRequestWithContext(ctx, "GET", "https://api.bilibili.com/x/player/playurl?"+params.Encode(), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	resp, e = client.Do(req)
	if e != nil {
		return biliVideo{}, e
	}
	defer resp.Body.Close()
	var play struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			DURL []struct {
				URL  string `json:"url"`
				Size int64  `json:"size"`
			} `json:"durl"`
		} `json:"data"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&play) != nil || play.Code != 0 || len(play.Data.DURL) == 0 {
		return biliVideo{}, errors.New("无法获取 MP4 视频流")
	}
	maxBytes, maxSeconds := videoLimits()
	stream := play.Data.DURL[0]
	if stream.Size > maxBytes {
		return biliVideo{}, fmt.Errorf("视频超过 %d MB 限制", maxBytes>>20)
	}
	if view.Data.Duration > maxSeconds {
		return biliVideo{}, fmt.Errorf("视频超过 %d 秒限制", maxSeconds)
	}
	return biliVideo{view.Data.BVID, cid, view.Data.Title, view.Data.Duration, stream.Size, stream.URL}, nil
}
func downloadBili(ctx context.Context, v biliVideo) (string, error) {
	maxBytes, _ := videoLimits()
	dir := filepath.Join("data", "bilibili")
	if e := os.MkdirAll(dir, 0755); e != nil {
		return "", e
	}
	f, e := os.CreateTemp(dir, "bili-*.mp4")
	if e != nil {
		return "", e
	}
	path := f.Name()
	ok := false
	defer func() {
		f.Close()
		if !ok {
			os.Remove(path)
		}
	}()
	req, e := http.NewRequestWithContext(ctx, "GET", v.URL, nil)
	if e != nil {
		return "", e
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	resp, e := (&http.Client{Timeout: 75 * time.Second}).Do(req)
	if e != nil {
		return "", e
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errors.New("视频下载接口返回错误")
	}
	if resp.ContentLength > maxBytes {
		return "", errors.New("视频文件超过大小限制")
	}
	n, e := io.Copy(f, io.LimitReader(resp.Body, maxBytes+1))
	if e != nil {
		return "", e
	}
	if n > maxBytes {
		return "", errors.New("下载视频超过大小限制")
	}
	if e = f.Close(); e != nil {
		return "", e
	}
	ok = true
	return path, nil
}

var _ = filepath.Base
var _ = onebot.Event{}
