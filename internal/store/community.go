package store

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

func (s *Store) RecordMessage(group, user, nickname, text string) error {
	if r := []rune(nickname); len(r) > 80 {
		nickname = string(r[:80])
	}
	if r := []rune(text); len(r) > 4000 {
		text = string(r[:4000])
	}
	_, e := s.DB.Exec(`INSERT INTO group_messages(group_id,user_id,nickname,message_text,created_at) VALUES(?,?,?,?,?)`, group, user, nickname, text, time.Now().Format("2006-01-02 15:04:05"))
	return e
}
func (s *Store) GroupStats(group string, days int) (int, int, error) {
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365
	}
	since := time.Now().AddDate(0, 0, -days+1).Format(`2006-01-02 00:00:00`)
	var m, u int
	e := s.DB.QueryRow(`SELECT COUNT(*),COUNT(DISTINCT user_id) FROM group_messages WHERE group_id=? AND created_at>=?`, group, since).Scan(&m, &u)
	return m, u, e
}
func (s *Store) GroupRanking(group string, days int) (string, error) {
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365
	}
	since := time.Now().AddDate(0, 0, -days+1).Format(`2006-01-02 00:00:00`)
	rows, e := s.DB.Query(`SELECT COALESCE(NULLIF(MAX(nickname),''),user_id),COUNT(*) n FROM group_messages WHERE group_id=? AND created_at>=? GROUP BY user_id ORDER BY n DESC,MAX(id) ASC LIMIT 10`, group, since)
	if e != nil {
		return "", e
	}
	defer rows.Close()
	out := "发言排行（最近 " + fmt.Sprint(days) + " 天）："
	i := 0
	for rows.Next() {
		var n string
		var c int
		if e = rows.Scan(&n, &c); e != nil {
			return "", e
		}
		i++
		out += fmt.Sprintf("\n%d. %s：%d 条", i, n, c)
	}
	if i == 0 {
		return fmt.Sprintf("最近 %d 天暂时没有统计到群消息。", days), rows.Err()
	}
	return out, rows.Err()
}

var urlRE = regexp.MustCompile("https?://\\S+|\\[CQ:[^\\]]+\\]")

func (s *Store) WordCounts(group string, days int) (map[string]int, error) {
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365
	}
	since := time.Now().AddDate(0, 0, -days+1).Format(`2006-01-02 00:00:00`)
	rows, e := s.DB.Query(`SELECT message_text FROM group_messages WHERE group_id=? AND created_at>=?`, group, since)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	counts := map[string]int{}
	stop := map[string]bool{"我们": true, "你我": true, "他们": true, "这个": true, "那个": true, "一个": true, "没有": true, "可以": true, "还是": true, "但是": true, "然后": true, "以及": true, "不是": true, "什么": true, "怎么": true, "真的": true, "现在": true, "进行": true, "回复": true, "你好": true, "哈哈": true, "呵呵": true}
	re := regexp.MustCompile("[A-Za-z][A-Za-z0-9_]{1,}|[\\x{4e00}-\\x{9fff}]{2,}")
	for rows.Next() {
		var text string
		if e = rows.Scan(&text); e != nil {
			return nil, e
		}
		trimmed := strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "!") || strings.HasPrefix(trimmed, "！") {
			continue
		}
		text = urlRE.ReplaceAllString(text, " ")
		for _, word := range re.FindAllString(text, -1) {
			if isChinese(word) {
				r := []rune(word)
				for i := 0; i < len(r)-1; i++ {
					part := string(r[i : i+2])
					if !stop[part] {
						counts[part]++
					}
				}
			} else {
				word = strings.ToLower(word)
				if !stop[word] {
					counts[word]++
				}
			}
		}
	}
	return counts, rows.Err()
}
func isChinese(s string) bool {
	for _, r := range s {
		if r < 0x4e00 || r > 0x9fff {
			return false
		}
	}
	return true
}
