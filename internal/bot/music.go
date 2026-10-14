package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type songResult struct {
	Platform           string
	ID                 int
	Title, Artist, URL string
}

func searchMusicResults(query, platform string) (string, []songResult) {
	var songs []songResult
	client := http.Client{Timeout: 8 * time.Second}
	search := func(endpoint string, params url.Values, parse func([]byte) string) string {
		req, _ := http.NewRequest("GET", endpoint+"?"+params.Encode(), nil)
		resp, err := client.Do(req)
		if err != nil {
			return ""
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return ""
		}
		var data json.RawMessage
		if json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&data) != nil {
			return ""
		}
		return parse(data)
	}
	out := ""
	if platform == "both" || platform == "qq" {
		p := url.Values{"w": {query}, "p": {"1"}, "n": {"10"}, "format": {"json"}}
		out = search("https://c.y.qq.com/soso/fcgi-bin/client_search_cp", p, func(b []byte) string {
			var x struct {
				Data struct {
					Song struct {
						List []struct {
							ID     int    `json:"songid"`
							MID    string `json:"songmid"`
							Title  string `json:"songname"`
							Singer []struct {
								Name string `json:"name"`
							} `json:"singer"`
						} `json:"list"`
					} `json:"song"`
				} `json:"data"`
			}
			if json.Unmarshal(b, &x) != nil || len(x.Data.Song.List) == 0 {
				return ""
			}
			best, score := 0, -1
			for i, song := range x.Data.Song.List {
				artist := ""
				if len(song.Singer) > 0 {
					artist = song.Singer[0].Name
				}
				if value := songMatchScore(query, song.Title, artist); value > score {
					best, score = i, value
				}
			}
			s := x.Data.Song.List[best]
			if s.ID <= 0 {
				return ""
			}
			artist := "未知歌手"
			if len(s.Singer) > 0 {
				artist = s.Singer[0].Name
			}
			detail := fmt.Sprintf("https://y.qq.com/n/ryqq/songDetail/%s", s.MID)
			songs = append(songs, songResult{"qq", s.ID, s.Title, artist, detail})
			return fmt.Sprintf("\nQQ 音乐：%s - %s\n%s", s.Title, artist, detail)
		})
	}
	if platform == "both" || platform == "163" {
		p := url.Values{"s": {query}, "type": {"1"}, "offset": {"0"}, "limit": {"10"}}
		out += search("https://music.163.com/api/search/get/web", p, func(b []byte) string {
			var x struct {
				Result struct {
					Songs []struct {
						ID      int    `json:"id"`
						Name    string `json:"name"`
						Artists []struct {
							Name string `json:"name"`
						} `json:"artists"`
					} `json:"songs"`
				} `json:"result"`
			}
			if json.Unmarshal(b, &x) != nil || len(x.Result.Songs) == 0 {
				return ""
			}
			best, score := 0, -1
			for i, song := range x.Result.Songs {
				artist := ""
				if len(song.Artists) > 0 {
					artist = song.Artists[0].Name
				}
				if value := songMatchScore(query, song.Name, artist); value > score {
					best, score = i, value
				}
			}
			s := x.Result.Songs[best]
			if s.ID <= 0 {
				return ""
			}
			artist := "未知歌手"
			if len(s.Artists) > 0 {
				artist = s.Artists[0].Name
			}
			detail := fmt.Sprintf("https://music.163.com/#/song?id=%d", s.ID)
			songs = append(songs, songResult{"163", s.ID, s.Name, artist, detail})
			return fmt.Sprintf("\n网易云音乐：%s - %s\n%s", s.Name, artist, detail)
		})
	}
	if out == "" {
		return "没有找到歌曲，或音乐平台暂时不可用。", nil
	}
	return "为你搜索「" + query + "」：" + out, songs
}

var songNormalizeRE = regexp.MustCompile(`[^0-9a-z\x{4e00}-\x{9fff}]`)

func songMatchScore(query, title, artist string) int {
	norm := func(s string) string { return songNormalizeRE.ReplaceAllString(strings.ToLower(s), "") }
	t, a := norm(title), norm(artist)
	score := 0
	for _, word := range strings.Fields(query) {
		w := norm(word)
		if w == "" {
			continue
		}
		if strings.Contains(t, w) {
			score += 2
		}
		if strings.Contains(a, w) {
			score += 5
		}
		if t == w {
			score += 8
		}
	}
	if t != "" && t == norm(query) {
		score += 10
	}
	return score
}
