package onebot

import (
	"encoding/json"
	"testing"
)

func TestCQAndArrayNormalization(t *testing.T) {
	for _, raw := range []string{`"[CQ:at,qq=99]hello &amp; &#91;x&#93;[CQ:image,file=a&#44;b,url=https://example.com/a.png]"`, `[{"type":"at","data":{"qq":"99"}},{"type":"text","data":{"text":"hello & [x]"}},{"type":"image","data":{"file":"a,b","url":"https://example.com/a.png"}}]`} {
		var e Event
		data := []byte(`{"post_type":"message","message":` + raw + `}`)
		if err := json.Unmarshal(data, &e); err != nil {
			t.Fatal(err)
		}
		if e.PlainText() != "hello & [x]" {
			t.Fatal(e.PlainText())
		}
		segs := ParseMessage(e.Message)
		if len(segs) != 3 || segs[2].Data["file"] != "a,b" {
			t.Fatal(segs)
		}
	}
	s := ParseMessage(json.RawMessage(`"[at:qq=123] hi"`))
	if s[0].Data["qq"] != "123" {
		t.Fatal(s)
	}
}
