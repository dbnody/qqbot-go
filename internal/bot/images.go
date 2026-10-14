package bot

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"qqbot-go/internal/onebot"
)

func memeTemplateList() string {
	return "Meme 模板列表：哭泣猫猫、完美、神图、PornHub标题图、举牌小人、熊猫人、狗头、震惊、经典、彩虹\n用法：/meme 模板 文字"
}

func (h *Handler) makeMeme(text string) (string, error) {
	face := fontFace(40)
	if face == nil {
		return "", fmt.Errorf("中文字体不可用，请配置 QQBOT_FONT_PATH")
	}
	face.Close()
	template, body, ok := strings.Cut(strings.TrimSpace(text), " ")
	if !ok {
		body, template = template, "经典"
	}
	aliases := map[string]string{"哭猫": "哭泣猫猫", "猫猫": "哭泣猫猫", "举牌": "举牌小人", "pornhub": "PornHub标题图", "pornhub标题图": "PornHub标题图"}
	if v, ok := aliases[strings.ToLower(template)]; ok {
		template = v
	}
	if _, ok := memeTemplates[template]; !ok {
		template = "经典"
	}
	body = truncate(strings.TrimSpace(body), 120)
	if body == "" {
		body = "今天也要开心喵"
	}
	img := renderMeme(template, body)
	dir := filepath.Join("data", "meme")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, "meme-*.png")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if err = png.Encode(f, img); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err = f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

var memeTemplates = map[string]struct{}{
	"完美": {}, "神图": {}, "PornHub标题图": {}, "pornhub": {}, "举牌小人": {}, "熊猫人": {},
	"狗头": {}, "震惊": {}, "彩虹": {}, "哭泣猫猫": {}, "哭猫": {}, "猫猫": {}, "经典": {},
}

func (h *Handler) wordCloud(group string, days int) (string, error) {
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365
	}
	face := fontFace(40)
	if face == nil {
		return "", fmt.Errorf("中文字体不可用，请配置 QQBOT_FONT_PATH")
	}
	face.Close()
	counts, err := h.Store.WordCounts(group, days)
	if err != nil {
		return "", err
	}
	if len(counts) == 0 {
		return "", nil
	}
	type item struct {
		word string
		n    int
	}
	items := []item{}
	for w, n := range counts {
		items = append(items, item{w, n})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].n == items[j].n {
			return items[i].word < items[j].word
		}
		return items[i].n > items[j].n
	})
	if len(items) > 45 {
		items = items[:45]
	}
	img := image.NewRGBA(image.Rect(0, 0, 900, 600))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{247, 250, 255, 255}}, image.Point{}, draw.Src)
	borderedRect(img, img.Bounds(), nil, color.RGBA{74, 105, 160, 255}, 3)
	drawText(img, fmt.Sprintf("群聊词云 · 最近 %d 天", days), 28, 48, color.RGBA{35, 54, 95, 255})
	max := items[0].n
	colors := []color.RGBA{{42, 91, 145, 255}, {40, 130, 113, 255}, {194, 94, 62, 255}, {119, 78, 154, 255}, {180, 123, 36, 255}}
	rng := rand.New(rand.NewSource(int64(groupHash(group + time.Now().Format("2006-01-02")))))
	placed := []image.Rectangle{}
	for i, it := range items {
		size := 22 + int(46*float64(it.n)/float64(max))
		if size > 68 {
			size = 68
		}
		w := textWidth(it.word, size)
		h := size + 8
		for tries := 0; tries < 80; tries++ {
			x := 25
			if 875-w > 25 {
				x += rng.Intn(875 - w - 25)
			}
			y := 85
			if 565-h > 85 {
				y += rng.Intn(565 - h - 85)
			}
			box := image.Rect(x-4, y-3, x+w+4, y+h+3)
			overlap := false
			for _, old := range placed {
				if box.Overlaps(old) {
					overlap = true
					break
				}
			}
			if !overlap {
				drawTextSize(img, it.word, x, y+size, colors[i%len(colors)], size)
				placed = append(placed, box)
				break
			}
		}
	}
	dir := filepath.Join("data", "wordcloud")
	if err = os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, "cloud-*.png")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if err = png.Encode(f, img); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err = f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}
func groupHash(s string) uint32 {
	var h uint32 = 2166136261
	for _, r := range s {
		h ^= uint32(r)
		h *= 16777619
	}
	return h
}

var parsedFonts sync.Map

func fontFace(size float64) font.Face {
	candidates := []string{os.Getenv("QQBOT_FONT_PATH"), "C:/Windows/Fonts/simhei.ttf", "C:/Windows/Fonts/msyh.ttc", "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc", "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		var f *opentype.Font
		if cached, ok := parsedFonts.Load(p); ok {
			f = cached.(*opentype.Font)
		} else {
			b, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			f, err = opentype.Parse(b)
			if err != nil {
				collection, err := opentype.ParseCollection(b)
				if err != nil {
					continue
				}
				f, err = collection.Font(0)
				if err != nil {
					continue
				}
			}
			parsedFonts.Store(p, f)
		}
		face, e := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
		if e == nil {
			return face
		}
	}
	return nil
}
func drawTextSize(img *image.RGBA, text string, x, y int, c color.Color, size int) {
	face := fontFace(float64(size))
	if face == nil {
		return
	}
	defer face.Close()
	d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, y)}
	d.DrawString(text)
}
func drawText(img *image.RGBA, text string, x, y int, c color.Color) {
	drawTextSize(img, text, x, y, c, 42)
}
func textWidth(s string, size int) int {
	f := fontFace(float64(size))
	if f == nil {
		return len([]rune(s)) * size
	}
	defer f.Close()
	return font.MeasureString(f, s).Ceil()
}
func fillRect(img *image.RGBA, r image.Rectangle, c color.Color) {
	draw.Draw(img, r, &image.Uniform{C: c}, image.Point{}, draw.Src)
}
func fillCircle(img *image.RGBA, cx, cy, r int, col color.Color) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				px, py := cx+x, cy+y
				if image.Pt(px, py).In(img.Bounds()) {
					img.Set(px, py, col)
				}
			}
		}
	}
}
func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if image.Pt(x0, y0).In(img.Bounds()) {
			img.Set(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}
func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
func (h *Handler) sendImage(e onebot.Event, path, caption string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	message := []onebot.Segment{}
	if caption != "" {
		message = append(message, onebot.Segment{Type: "text", Data: map[string]any{"text": caption}})
	}
	message = append(message, onebot.Segment{Type: "image", Data: map[string]any{"file": "base64://" + base64.StdEncoding.EncodeToString(data)}})
	params := map[string]any{"message": message}
	if e.MessageType == "group" {
		params["group_id"] = e.GroupID
	} else {
		params["user_id"] = e.UserID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err = h.Client.Call(ctx, "send_msg", params)
	return err
}
