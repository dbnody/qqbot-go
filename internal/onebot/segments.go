package onebot

import (
	"encoding/json"
	"regexp"
	"strings"
)

type Segment struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

var cqRE = regexp.MustCompile(`\[CQ:([A-Za-z_]+)((?:,[^\]]*)?)\]|\[at:qq=([0-9]+)\]`)

func cqDecode(s string, parameter bool) string {
	if parameter {
		s = strings.ReplaceAll(s, "&#44;", ",")
	}
	return strings.NewReplacer("&#91;", "[", "&#93;", "]", "&amp;", "&").Replace(s)
}
func ParseMessage(raw json.RawMessage) []Segment {
	var segments []Segment
	if json.Unmarshal(raw, &segments) == nil {
		return segments
	}
	var text string
	if json.Unmarshal(raw, &text) != nil {
		return nil
	}
	last := 0
	for _, m := range cqRE.FindAllStringSubmatchIndex(text, -1) {
		if m[0] > last {
			segments = append(segments, Segment{"text", map[string]any{"text": cqDecode(text[last:m[0]], false)}})
		}
		typ := "at"
		data := map[string]any{}
		if m[6] >= 0 {
			data["qq"] = text[m[6]:m[7]]
		} else {
			typ = text[m[2]:m[3]]
			for _, part := range strings.Split(strings.TrimPrefix(text[m[4]:m[5]], ","), ",") {
				key, value, ok := strings.Cut(part, "=")
				if ok {
					data[key] = cqDecode(value, true)
				}
			}
		}
		segments = append(segments, Segment{typ, data})
		last = m[1]
	}
	if last < len(text) {
		segments = append(segments, Segment{"text", map[string]any{"text": cqDecode(text[last:], false)}})
	}
	return segments
}
