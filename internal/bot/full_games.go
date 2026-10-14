package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"qqbot-go/internal/onebot"
	"strconv"
	"strings"
	"sync"
)

type rouletteState struct {
	Owner           string
	Players         []string
	Names           map[string]string
	Bullet, Chamber int
	Started, Solo   bool
}
type blackjackState struct {
	Players      []string
	Names        map[string]string
	Hands        map[string][]string
	Deck, Dealer []string
	Turn         int
	Stood        map[string]bool
	Started      bool
}
type socialState struct {
	Kind, Answer, Hint, Last, Drawer string
	HasImage                         bool
	Artist                           string
}

var gamesMu sync.Mutex
var rouletteGames = map[string]*rouletteState{}
var blackjackGames = map[string]*blackjackState{}
var socialGames = map[string]*socialState{}
var idiomSet = map[string]bool{"一心一意": true, "意气风发": true, "发扬光大": true, "大展宏图": true, "图穷匕见": true, "见多识广": true, "广开言路": true, "路不拾遗": true, "遗臭万年": true, "年富力强": true, "强词夺理": true, "理直气壮": true, "壮志凌云": true, "云开见日": true, "日新月异": true, "异想天开": true, "开门见山": true, "山清水秀": true, "秀外慧中": true, "中流砥柱": true, "柱石之坚": true, "坚韧不拔": true, "拔苗助长": true, "长驱直入": true, "入木三分": true, "分秒必争": true, "争先恐后": true, "后来居上": true, "上下一心": true, "心想事成": true}

func gameKey(e onebot.Event) string {
	if e.MessageType == "group" {
		return "g:" + strings.Trim(string(e.GroupID), "\"")
	}
	return "p:" + strings.Trim(string(e.UserID), "\"")
}
func (h *Handler) fullGameCommand(name, args string, e onebot.Event, nickname string) string {
	key := gameKey(e)
	uid := strings.Trim(string(e.UserID), "\"")
	action := strings.ToLower(strings.TrimSpace(args))
	gamesMu.Lock()
	defer gamesMu.Unlock()
	if name == "轮盘赌" || name == "roulette" || name == "俄罗斯轮盘" || name == "左轮赌" {
		g := rouletteGames[key]
		switch action {
		case "", "帮助", "help":
			return "左轮轮盘赌：/轮盘赌 单人；多人房主 /轮盘赌 多人，其他人加入，房主开始；当前玩家 /轮盘赌 扣扳机。中弹后群聊将尝试禁言60秒。"
		case "单人", "自己玩", "solo":
			rouletteGames[key] = &rouletteState{Owner: uid, Players: []string{uid}, Names: map[string]string{uid: nickname}, Bullet: rand.Intn(6), Started: true, Solo: true}
			return "左轮已装填一发子弹。单人游戏开始，请发送 /轮盘赌 扣扳机。"
		case "多人", "多人玩", "多人游戏", "multi":
			if g != nil && g.Started {
				return "当前已经有一局进行中，请等待本局结束。"
			}
			rouletteGames[key] = &rouletteState{Owner: uid, Players: []string{uid}, Names: map[string]string{uid: nickname}, Bullet: rand.Intn(6)}
			return "多人房间已创建。发送 /轮盘赌 加入，房主发送 /轮盘赌 开始。"
		}
		if g == nil {
			return "还没有轮盘赌房间，请先发送 /轮盘赌 单人 或 /轮盘赌 多人。"
		}
		switch action {
		case "加入", "join":
			if g.Started {
				return "本局已经开始，不能中途加入。"
			}
			for _, p := range g.Players {
				if p == uid {
					return "你已经在房间里了。"
				}
			}
			if len(g.Players) >= 6 {
				return "左轮最多支持 6 名玩家。"
			}
			g.Players = append(g.Players, uid)
			g.Names[uid] = nickname
			return fmt.Sprintf("%s 加入房间，当前人数：%d。", nickname, len(g.Players))
		case "开始", "start":
			if g.Started {
				return "本局已经开始了。"
			}
			if uid != g.Owner {
				return "只有房主可以开始游戏。"
			}
			if len(g.Players) < 2 {
				return "多人游戏至少需要 2 名玩家。"
			}
			g.Started = true
			return "左轮已装填一发子弹，轮流扣扳机。现在轮到 " + g.Names[g.Players[0]] + "。"
		case "状态", "信息", "status":
			names := []string{}
			for _, p := range g.Players {
				names = append(names, g.Names[p])
			}
			if g.Started {
				return fmt.Sprintf("当前玩家：%s\n玩家列表：%s\n已扣扳机：%d 次", g.Names[g.Players[g.Chamber%len(g.Players)]], strings.Join(names, "、"), g.Chamber)
			}
			return "等待开始，玩家列表：" + strings.Join(names, "、")
		case "退出", "结束", "quit":
			if uid == g.Owner {
				delete(rouletteGames, key)
				return "本局轮盘赌已结束。"
			}
			for i, p := range g.Players {
				if p == uid {
					g.Players = append(g.Players[:i], g.Players[i+1:]...)
					delete(g.Names, uid)
					return "你已退出当前房间。"
				}
			}
			return "你不在当前房间。"
		}
		if action != "扣扳机" && action != "开枪" && action != "开火" && action != "trigger" && action != "go" {
			return "请使用：/轮盘赌 加入、开始、扣扳机、状态或退出。"
		}
		if !g.Started {
			return "多人房间还未开始，请先加入后由房主发送 /轮盘赌 开始。"
		}
		current := g.Players[g.Chamber%len(g.Players)]
		if current != uid {
			return "还没轮到你，请等待 " + g.Names[current] + "。"
		}
		currentName := g.Names[uid]
		if g.Chamber == g.Bullet {
			delete(rouletteGames, key)
			muted := false
			if e.MessageType == "group" {
				if h.Client != nil {
					_, err := h.Client.Call(context.Background(), "set_group_ban", map[string]any{"group_id": e.GroupID, "user_id": e.UserID, "duration": 60})
					muted = err == nil
				}
			}
			punish := "当前为私聊或 Bot 没有群管理权限，未能执行禁言"
			if muted {
				punish = "已禁言 60 秒"
			}
			return fmt.Sprintf("砰！%s 中弹了！本局结束，%s。", currentName, punish)
		}
		g.Chamber++
		next := g.Players[g.Chamber%len(g.Players)]
		return fmt.Sprintf("咔哒……%s 幸存！下一位是 %s。", currentName, g.Names[next])
	}
	if name == "21点" || name == "21" || name == "二十一点" || name == "blackjack" {
		g := blackjackGames[key]
		switch action {
		case "创建", "多人", "新建", "create":
			blackjackGames[key] = &blackjackState{Names: map[string]string{}, Hands: map[string][]string{}, Stood: map[string]bool{}}
			return "已创建多人 21 点房间。其他玩家使用 /21点 加入，人数足够后使用 /21点 开始。"
		case "加入", "join":
			if g == nil {
				return "还没有房间，请先使用 /21点 创建。"
			}
			if g.Started {
				return "21 点已经开始，不能中途加入。"
			}
			for _, p := range g.Players {
				if p == uid {
					return nickname + " 已经加入了。"
				}
			}
			if len(g.Players) >= 6 {
				return "多人 21 点最多 6 名玩家。"
			}
			g.Players = append(g.Players, uid)
			g.Names[uid] = nickname
			return fmt.Sprintf("%s 加入成功，当前玩家：%s", nickname, strings.Join(blackjackNames(g), "、"))
		case "开始", "start":
			if g == nil {
				return "还没有房间，请先使用 /21点 创建。"
			}
			if g.Started {
				return "21 点已经开始。"
			}
			if len(g.Players) < 2 {
				return "多人 21 点至少需要 2 名玩家。"
			}
			g.Deck = makeDeck()
			g.Hands = map[string][]string{}
			for _, p := range g.Players {
				g.Hands[p] = []string{drawCard(g), drawCard(g)}
			}
			g.Dealer = []string{drawCard(g), drawCard(g)}
			g.Turn = 0
			g.Stood = map[string]bool{}
			g.Started = true
			return fmt.Sprintf("21 点开始！玩家：%s\n庄家明牌：%s，当前轮到 %s。\n请使用 /21点 要牌 或 /21点 停牌。", strings.Join(blackjackNames(g), "、"), g.Dealer[0], g.Names[g.Players[0]])
		case "状态", "查看", "status":
			if g == nil {
				return "当前没有 21 点房间。"
			}
			return blackjackStatus(g)
		}
		if g == nil {
			return "还没有进行中的 21 点。"
		}
		if action == "退出" || action == "结束" || action == "reset" {
			delete(blackjackGames, key)
			return "21 点房间已结束。"
		}
		if !g.Started {
			return "当前没有进行中的 21 点。"
		}
		if g.Turn >= len(g.Players) {
			return "本局即将结算。"
		}
		current := g.Players[g.Turn]
		if current != uid {
			return "还没轮到你，请等待 " + g.Names[current] + "。"
		}
		if action == "要牌" || action == "要" || action == "hit" {
			card := drawCard(g)
			g.Hands[uid] = append(g.Hands[uid], card)
			score := handScore(g.Hands[uid])
			msg := fmt.Sprintf("%s 要牌：%s，当前点数 %d。", g.Names[uid], card, score)
			if score > 21 {
				g.Stood[uid] = true
				msg += " 爆牌，自动停牌。"
			} else if score == 21 {
				g.Stood[uid] = true
				msg += " 21 点，自动停牌。"
			} else {
				return msg
			}
			return msg + "\n" + advanceBlackjack(g)
		}
		if action == "停牌" || action == "停" || action == "stand" {
			g.Stood[uid] = true
			return g.Names[uid] + " 选择停牌。\n" + advanceBlackjack(g)
		}
		return "用法：/21点 创建、加入、开始、要牌、停牌、状态、退出。"
	}
	return h.socialCommand(name, args, e, nickname)
}
func blackjackNames(g *blackjackState) []string {
	v := []string{}
	for _, id := range g.Players {
		v = append(v, g.Names[id])
	}
	return v
}
func makeDeck() []string {
	suits := []string{"♠", "♥", "♦", "♣"}
	ranks := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	d := []string{}
	for _, s := range suits {
		for _, r := range ranks {
			d = append(d, s+r)
		}
	}
	rand.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
	return d
}
func drawCard(g *blackjackState) string {
	i := len(g.Deck) - 1
	c := g.Deck[i]
	g.Deck = g.Deck[:i]
	return c
}
func cardValue(c string) int {
	r := strings.TrimLeft(c, "♠♥♦♣")
	switch r {
	case "A":
		return 11
	case "J", "Q", "K":
		return 10
	default:
		n, _ := strconv.Atoi(r)
		return n
	}
}
func handScore(h []string) int {
	v, a := 0, 0
	for _, c := range h {
		v += cardValue(c)
		if strings.HasSuffix(c, "A") {
			a++
		}
	}
	for v > 21 && a > 0 {
		v -= 10
		a--
	}
	return v
}
func advanceBlackjack(g *blackjackState) string {
	for g.Turn < len(g.Players) && (g.Stood[g.Players[g.Turn]] || handScore(g.Hands[g.Players[g.Turn]]) > 21) {
		g.Turn++
	}
	if g.Turn < len(g.Players) {
		return "轮到 " + g.Names[g.Players[g.Turn]] + "，请使用 /21点 要牌 或 /21点 停牌。"
	}
	for handScore(g.Dealer) < 17 {
		g.Dealer = append(g.Dealer, drawCard(g))
	}
	ds := handScore(g.Dealer)
	lines := []string{fmt.Sprintf("庄家手牌：%s，点数：%d", " "+strings.Join(g.Dealer, " "), ds), "本局结算："}
	for _, p := range g.Players {
		s := handScore(g.Hands[p])
		res := "失败"
		if s > 21 {
			res = "爆牌，输"
		} else if ds > 21 || s > ds {
			res = "胜利"
		} else if s == ds {
			res = "平局"
		}
		lines = append(lines, fmt.Sprintf("%s：%s（%d 点，%s）", g.Names[p], strings.Join(g.Hands[p], " "), s, res))
	}
	g.Started = false
	return strings.Join(lines, "\n") + "\n使用 /21点 创建 开始新一局。"
}
func blackjackStatus(g *blackjackState) string {
	lines := []string{"21 点房间（等待开始）", "玩家：" + strings.Join(blackjackNames(g), "、")}
	if g.Started {
		lines[0] = "21 点房间（进行中）"
		lines = append(lines, "庄家明牌："+g.Dealer[0])
		for _, p := range g.Players {
			lines = append(lines, fmt.Sprintf("%s：%s（%d 点）", g.Names[p], strings.Join(g.Hands[p], " "), handScore(g.Hands[p])))
		}
		if g.Turn < len(g.Players) {
			lines = append(lines, "当前轮次："+g.Names[g.Players[g.Turn]])
		}
	}
	return strings.Join(lines, "\n")
}
func (h *Handler) socialCommand(name, args string, e onebot.Event, nickname string) string {
	if e.MessageType != "group" {
		return "这个小游戏只能在群聊中使用。"
	}
	key := gameKey(e)
	a := strings.TrimSpace(args)
	if a == "结束" || a == "停止" || a == "end" {
		gamesMu.Lock()
		delete(socialGames, key)
		gamesMu.Unlock()
		return "本轮游戏已结束。"
	}
	gamesMu.Lock()
	defer gamesMu.Unlock()
	g := socialGames[key]
	switch name {
	case "猜谜", "猜谜语", "谜语":
		if a == "提示" || a == "给提示" || a == "hint" {
			if g != nil && g.Kind == "riddle" {
				return "提示：" + g.Hint
			}
			return "当前没有进行中的猜谜。"
		}
		questions := catalog.Riddles
		x := questions[rand.Intn(len(questions))]
		socialGames[key] = &socialState{Kind: "riddle", Answer: x[1], Hint: x[2]}
		return "猜谜开始！\n" + x[0] + "\n直接发送答案，/猜谜 提示 查看提示。"
	case "成语接龙", "接龙", "idiom":
		start := catalog.Idioms[rand.Intn(len(catalog.Idioms))]
		socialGames[key] = &socialState{Kind: "idiom", Last: start}
		return fmt.Sprintf("成语接龙开始！\n第一个成语：%s\n下一位请接“%s”开头的成语。", start, string([]rune(start)[len([]rune(start))-1]))
	case "你画我猜", "画猜", "draw":
		choices := [][2]string{{"长城", "著名的中国建筑"}, {"大熊猫", "黑白相间的国宝"}, {"飞机", "可以在天空飞行"}, {"蛋糕", "生日时常见的甜点"}, {"吉他", "常见的弦乐器"}}
		x := choices[rand.Intn(len(choices))]
		uid := strings.Trim(string(e.UserID), "\"")
		socialGames[key] = &socialState{Kind: "draw", Answer: x[0], Hint: x[1], Drawer: uid}
		_, err := h.Client.Call(context.Background(), "send_private_msg", map[string]any{"user_id": e.UserID, "message": "你是本轮画手，答案是：" + x[0] + "\n提示：" + x[1] + "\n请在群里发送图片。"})
		if err != nil {
			return "你画我猜开始！暂时无法发送私聊，请画手确认已开启临时会话。"
		}
		return "你画我猜开始！画手答案已私聊发送，请画手发送图片，其他成员直接发送答案。"
	case "猜歌名", "猜歌", "听歌猜名", "song":
		if a == "提示" || a == "给提示" || a == "hint" {
			if g != nil && g.Kind == "song" {
				return "歌手提示：" + g.Artist
			}
			return "当前没有猜歌回合。"
		}
		songs := catalog.Songs
		x := songs[rand.Intn(len(songs))]
		socialGames[key] = &socialState{Kind: "song", Answer: x[0], Artist: x[1], Hint: x[2]}
		return "猜歌名开始！\n歌词提示：" + x[2] + "\n直接发送歌名作答，/猜歌名 提示 查看歌手提示。"
	}
	return "未知群游戏命令。"
}
func (h *Handler) handleSocialMessage(e onebot.Event, text string) string {
	if e.MessageType != "group" {
		return ""
	}
	key := gameKey(e)
	gamesMu.Lock()
	defer gamesMu.Unlock()
	g := socialGames[key]
	if g == nil {
		return ""
	}
	text = strings.TrimSpace(text)
	if text == "" {
		if g.Kind == "draw" && strings.Trim(string(e.UserID), `"`) == g.Drawer {
			var segments []struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(e.Message, &segments) == nil {
				for _, segment := range segments {
					if segment.Type == "image" {
						g.HasImage = true
						return "画作已收到！大家开始猜吧，直接发送你认为的答案。"
					}
				}
			}
		}
		return ""
	}
	if strings.HasPrefix(text, h.Prefix) {
		return ""
	}
	switch g.Kind {
	case "riddle":
		if text == "提示" || text == "给提示" {
			return "提示：" + g.Hint
		}
		if normalize(text) == normalize(g.Answer) {
			delete(socialGames, key)
			return "回答正确！答案是“" + g.Answer + "”，下一轮发送 /猜谜。"
		}
	case "idiom":
		r := []rune(text)
		last := []rune(g.Last)
		if len(r) < 4 {
			return ""
		}
		if !idiomSet[text] {
			return "这个成语不在本轮词库中，请换一个常见成语。"
		}
		if r[0] != last[len(last)-1] {
			return fmt.Sprintf("接龙失败，应使用“%s”开头的成语。", string(last[len(last)-1]))
		}
		g.Last = text
		return fmt.Sprintf("%s，接得好！下一位请接“%s”开头的成语。", text, string(r[len(r)-1]))
	case "song":
		if normalize(text) == normalize(g.Answer) {
			delete(socialGames, key)
			return fmt.Sprintf("猜对了！答案是《%s》——%s。下一轮发送 /猜歌名。", g.Answer, g.Artist)
		}
		if text == "提示" || text == "给提示" {
			return "歌手提示：" + g.Artist
		}
		return "还没猜对，再想想看～"
	case "draw":
		if g.HasImage && normalize(text) == normalize(g.Answer) {
			delete(socialGames, key)
			return "猜对了！答案是“" + g.Answer + "”，本轮你画我猜结束。"
		}
	}
	return ""
}
func normalize(s string) string {
	s = strings.ToLower(strings.Join(strings.Fields(s), ""))
	return strings.NewReplacer("，", "", "。", "", "！", "", "？", "", "!", "", "?", "", ",", "", ".", "", "、", "", "四", "4", "三", "3", "二", "2", "一", "1").Replace(s)
}

var _ = json.Valid
