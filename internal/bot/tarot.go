package bot

import (
	"crypto/rand"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/big"
	mathrand "math/rand"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"qqbot-go/internal/onebot"
)

type tarotCard struct{ Number, Name, Upright, Reversed, Keyword string }

var tarotDeck = []tarotCard{
	{"0", "愚者", "新的开始、自由、勇气", "鲁莽、准备不足、逃避责任", "相信旅程，但别忘了看路"},
	{"I", "魔术师", "行动力、创造力、资源整合", "分心、操控、能力未被使用", "你手中已经有足够的工具"},
	{"II", "女祭司", "直觉、内在智慧、等待时机", "忽视直觉、秘密、信息不足", "安静下来，答案会浮现"},
	{"III", "皇后", "丰盛、滋养、创造、成长", "过度依赖、停滞、忽略自己", "给正在成长的事物一点时间"},
	{"IV", "皇帝", "秩序、责任、领导力、稳定", "控制欲、僵化、权威冲突", "建立边界，再承担责任"},
	{"V", "教皇", "传统、学习、指引、信念", "固执、盲从、拒绝新观点", "向可靠的前辈或知识请教"},
	{"VI", "恋人", "选择、合作、吸引、价值一致", "犹豫、失衡、沟通不畅", "诚实面对自己的真实选择"},
	{"VII", "战车", "胜利、决心、推进、掌控方向", "失控、急躁、目标分散", "把力量集中到一个方向"},
	{"VIII", "力量", "耐心、勇气、温柔的坚持", "自我怀疑、压抑、情绪失控", "真正的力量不必大声证明"},
	{"IX", "隐者", "独处、反思、寻找答案", "孤立、逃避、拒绝帮助", "暂时慢下来并不等于落后"},
	{"X", "命运之轮", "转机、变化、周期、新机会", "抗拒变化、反复、时机未到", "顺势而为，留意意外机会"},
	{"XI", "正义", "公平、因果、清晰判断", "偏见、逃避后果、失衡", "用事实和长期影响做决定"},
	{"XII", "倒吊人", "换角度、暂停、放下执念", "拖延、无谓牺牲、停滞", "改变视角，可能看到新的出口"},
	{"XIII", "死神", "结束、转化、告别旧阶段", "抗拒改变、拖延结束、停留过去", "结束是为新阶段腾出空间"},
	{"XIV", "节制", "平衡、疗愈、协调、耐心", "失衡、过度、节奏混乱", "把不同的部分慢慢调和"},
	{"XV", "恶魔", "欲望、束缚、沉迷、现实诱惑", "挣脱、看见依赖、重新掌控", "确认什么在消耗你的自由"},
	{"XVI", "高塔", "突变、真相、旧结构瓦解", "逃避变化、延迟冲击、恐惧", "不稳的结构被打破后才能重建"},
	{"XVII", "星星", "希望、疗愈、灵感、信任未来", "失望、信心不足、能量低落", "保留希望，也给自己实际支持"},
	{"XVIII", "月亮", "潜意识、想象、未明信息", "迷雾散去、看清恐惧、误会解除", "先核实信息，再做结论"},
	{"XIX", "太阳", "成功、快乐、清晰、活力", "短暂低落、过度乐观、延迟", "坦率表达会带来温暖回应"},
	{"XX", "审判", "觉醒、复盘、重新选择", "自我否定、拒绝改变、错过召唤", "回顾过去，然后做出新的决定"},
	{"XXI", "世界", "完成、整合、圆满、新循环", "未完结、缺少收尾、重复旧课题", "给阶段性成果一个正式的结束"},
}

func secureIndex(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return mathrand.Intn(n)
	}
	return int(v.Int64())
}

func renderTarot(card tarotCard, reversed bool, userID string) (string, error) {
	face := fontFace(30)
	if face == nil {
		return "", fmt.Errorf("中文字体不可用，请设置 QQBOT_FONT_PATH")
	}
	face.Close()
	img := image.NewRGBA(image.Rect(0, 0, 720, 960))
	fillRect(img, img.Bounds(), color.RGBA{247, 237, 211, 255})
	gold := color.RGBA{173, 119, 38, 255}
	ink := color.RGBA{47, 37, 56, 255}
	fillRect(img, image.Rect(17, 17, 703, 25), gold)
	fillRect(img, image.Rect(17, 935, 703, 943), gold)
	fillRect(img, image.Rect(17, 17, 25, 943), gold)
	fillRect(img, image.Rect(695, 17, 703, 943), gold)
	fillRect(img, image.Rect(38, 38, 682, 40), ink)
	fillRect(img, image.Rect(38, 920, 682, 922), ink)
	fillRect(img, image.Rect(38, 38, 40, 922), ink)
	fillRect(img, image.Rect(680, 38, 682, 922), ink)
	center := func(text string, y, size int, c color.Color) {
		f := fontFace(float64(size))
		if f == nil {
			return
		}
		defer f.Close()
		w := font.MeasureString(f, text).Ceil()
		d := font.Drawer{Dst: img, Src: image.NewUniform(c), Face: f, Dot: fixed.P((720-w)/2, y+size)}
		d.DrawString(text)
	}
	center("TAROT · 今日启示", 65, 28, gold)
	center(card.Number+" · "+card.Name, 125, 48, ink)
	position := "正位"
	posColor := gold
	if reversed {
		position = "逆位"
		posColor = color.RGBA{142, 67, 82, 255}
	}
	center(position, 195, 30, posColor)
	ellipse(img, image.Rect(170, 250, 550, 630), nil, gold, 5)
	ellipse(img, image.Rect(190, 270, 530, 610), nil, ink, 2)
	if reversed {
		fillTriangle(img, [3]image.Point{{360, 560}, {265, 370}, {455, 370}}, color.RGBA{224, 191, 160, 255}, ink)
	} else {
		fillTriangle(img, [3]image.Point{{360, 300}, {265, 490}, {455, 490}}, color.RGBA{224, 191, 160, 255}, ink)
	}
	ellipse(img, image.Rect(305, 390, 415, 500), nil, gold, 5)
	drawLine(img, 250, 440, 470, 440, gold)
	drawLine(img, 360, 330, 360, 550, gold)
	center("关键词："+card.Keyword, 655, 25, ink)
	meaning := card.Upright
	if reversed {
		meaning = card.Reversed
	}
	lines := wrapRunes("牌面提示："+meaning, 24)
	for i, line := range lines {
		if i == 3 {
			break
		}
		center(line, 720+i*38, 24, ink)
	}
	center("仅供娱乐与自我反思", 855, 20, color.RGBA{100, 91, 100, 255})
	dir := filepath.Join("data", "tarot")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, "tarot-*.png")
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

func fillTriangle(img *image.RGBA, p [3]image.Point, fill, outline color.Color) {
	minX, maxX := p[0].X, p[0].X
	minY, maxY := p[0].Y, p[0].Y
	for _, q := range p[1:] {
		if q.X < minX {
			minX = q.X
		}
		if q.X > maxX {
			maxX = q.X
		}
		if q.Y < minY {
			minY = q.Y
		}
		if q.Y > maxY {
			maxY = q.Y
		}
	}
	cross := func(a, b, c image.Point) int { return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X) }
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			q := image.Pt(x, y)
			a, b, c := cross(p[0], p[1], q), cross(p[1], p[2], q), cross(p[2], p[0], q)
			if (a >= 0 && b >= 0 && c >= 0) || (a <= 0 && b <= 0 && c <= 0) {
				img.Set(x, y, fill)
			}
		}
	}
	drawLine(img, p[0].X, p[0].Y, p[1].X, p[1].Y, outline)
	drawLine(img, p[1].X, p[1].Y, p[2].X, p[2].Y, outline)
	drawLine(img, p[2].X, p[2].Y, p[0].X, p[0].Y, outline)
}
func wrapRunes(s string, n int) []string {
	r := []rune(strings.TrimSpace(s))
	var out []string
	for len(r) > 0 {
		k := n
		if len(r) < k {
			k = len(r)
		}
		out = append(out, string(r[:k]))
		r = r[k:]
	}
	return out
}

func (h *Handler) tarotCommand(e onebot.Event, args string) string {
	card := tarotDeck[secureIndex(len(tarotDeck))]
	reversed := secureIndex(2) == 1
	path, err := renderTarot(card, reversed, strings.Trim(string(e.UserID), `"`))
	if err != nil {
		return "塔罗图片生成失败：" + err.Error()
	}
	meaning := card.Upright
	position := "正位"
	if reversed {
		meaning = card.Reversed
		position = "逆位"
	}
	question := strings.TrimSpace(args)
	if question == "" {
		question = "没有特别指定的问题"
	}
	caption := fmt.Sprintf("你的问题：%s\n抽到：%s · %s（%s）\n关键词：%s\n解读：%s\n仅供娱乐与自我反思。", question, card.Number, card.Name, position, card.Keyword, meaning)
	if err = h.sendImage(e, path, caption); err != nil {
		_ = os.Remove(path)
		return "塔罗牌已生成，但图片发送失败，请检查 NapCat 图片路径配置。"
	}
	_ = os.Remove(path)
	return ""
}
