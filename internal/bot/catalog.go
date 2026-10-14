package bot

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed catalog.json
var catalogJSON []byte
var catalog struct {
	Quotes     []string    `json:"QUOTES"`
	Riddles    [][3]string `json:"RIDDLES"`
	Idioms     []string    `json:"IDIOMS"`
	Songs      [][3]string `json:"SONGS"`
	Origins    [][2]string `json:"_ORIGINS"`
	Talents    []string    `json:"_TALENTS"`
	Encounters []string    `json:"_ENCOUNTERS"`
	Endings    []string    `json:"_ENDINGS"`
	Stories    []string    `json:"_STORIES"`
}

func init() {
	if err := json.Unmarshal(catalogJSON, &catalog); err != nil {
		panic(err)
	}
	idiomSet = map[string]bool{}
	for _, s := range catalog.Idioms {
		idiomSet[s] = true
	}
}

var quotesMu sync.Mutex
var unusedQuotes = map[string][]int{}

func nextQuote(scope string) string {
	quotesMu.Lock()
	defer quotesMu.Unlock()
	remaining := unusedQuotes[scope]
	if len(remaining) == 0 {
		for i := range catalog.Quotes {
			remaining = append(remaining, i)
		}
	}
	i := secureIndex(len(remaining))
	quote := catalog.Quotes[remaining[i]]
	remaining[i] = remaining[len(remaining)-1]
	unusedQuotes[scope] = remaining[:len(remaining)-1]
	return quote
}
func rebornStory(n string) string {
	o := catalog.Origins[secureIndex(len(catalog.Origins))]
	return fmt.Sprintf("【投胎模拟器｜%s 的新人生】\n投胎结果：%s\n\n你出生时，%s。\n你的天赋是：%s。\n\n少年时期，你一边适应身世，一边凭借这项天赋闯出了自己的小天地。你经历过失败，也曾因为害怕而错过机会，但每一次跌倒都让你更清楚自己想成为什么样的人。\n\n人生转折：%s\n\n你活到了 %d 岁。%s\n\n本次投胎评价：人生没有标准答案，能把自己的故事过完，就是隐藏结局。", n, o[0], o[1], catalog.Talents[secureIndex(len(catalog.Talents))], catalog.Encounters[secureIndex(len(catalog.Encounters))], 52+secureIndex(66), catalog.Endings[secureIndex(len(catalog.Endings))])
}
