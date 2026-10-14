package bot

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"
)

// CPython's integer-seeded MT19937 keeps today's fortune unchanged on migration.
type pythonRandom struct {
	state [624]uint32
	index int
}

func newPythonRandom(seed [32]byte) *pythonRandom {
	r := &pythonRandom{index: 624}
	r.state[0] = 19650218
	for i := 1; i < 624; i++ {
		r.state[i] = 1812433253*(r.state[i-1]^(r.state[i-1]>>30)) + uint32(i)
	}
	var keys [8]uint32
	for i := range keys {
		keys[i] = binary.BigEndian.Uint32(seed[28-i*4 : 32-i*4])
	}
	i, j := 1, 0
	for k := 0; k < 624; k++ {
		r.state[i] = (r.state[i] ^ ((r.state[i-1] ^ (r.state[i-1] >> 30)) * 1664525)) + keys[j] + uint32(j)
		i++
		j++
		if i >= 624 {
			r.state[0] = r.state[623]
			i = 1
		}
		if j >= len(keys) {
			j = 0
		}
	}
	for k := 0; k < 623; k++ {
		r.state[i] = (r.state[i] ^ ((r.state[i-1] ^ (r.state[i-1] >> 30)) * 1566083941)) - uint32(i)
		i++
		if i >= 624 {
			r.state[0] = r.state[623]
			i = 1
		}
	}
	r.state[0] = 0x80000000
	return r
}
func (r *pythonRandom) next() uint32 {
	if r.index >= 624 {
		for i := 0; i < 624; i++ {
			y := (r.state[i] & 0x80000000) | (r.state[(i+1)%624] & 0x7fffffff)
			r.state[i] = r.state[(i+397)%624] ^ (y >> 1)
			if y&1 != 0 {
				r.state[i] ^= 0x9908b0df
			}
		}
		r.index = 0
	}
	y := r.state[r.index]
	r.index++
	y ^= y >> 11
	y ^= (y << 7) & 0x9d2c5680
	y ^= (y << 15) & 0xefc60000
	y ^= y >> 18
	return y
}
func (r *pythonRandom) below(n int) int {
	bits := 0
	for v := n; v > 0; v >>= 1 {
		bits++
	}
	for {
		value := int(r.next() >> uint(32-bits))
		if value < n {
			return value
		}
	}
}
func fortuneForDate(date, scope, nickname string) string {
	r := newPythonRandom(sha256.Sum256([]byte(date + ":" + scope)))
	levels := []string{"大吉", "吉", "中吉", "平"}
	advice := []string{"今天适合主动推进重要事项，容易遇到愿意帮助你的人。", "今天整体顺利，保持耐心，按计划完成一件重要的小事。", "今天有小惊喜，但不要急于求成，稳步行动会更有收获。", "今天适合整理和休息，先把手边的事情做好，再考虑下一步。"}
	colors := []string{"青绿色", "宝蓝色", "暖黄色", "紫色", "米白色", "橙色"}
	level := r.below(4)
	score := 60 + r.below(39)
	color := colors[r.below(6)]
	number := 1 + r.below(99)
	return fmt.Sprintf("%s 的今日运势（%s）\n综合运势：%s（%d分）\n幸运颜色：%s\n幸运数字：%d\n建议：%s\n\n仅供娱乐与自我激励。", nickname, date, levels[level], score, color, number, advice[level])
}
func dailyFortune(scope, nickname string) string {
	return fortuneForDate(time.Now().Format("2006-01-02"), scope, nickname)
}
