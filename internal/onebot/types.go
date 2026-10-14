package onebot

import (
	"encoding/json"
	"strings"
)

type Event struct {
	Time        int64           `json:"time"`
	SelfID      json.RawMessage `json:"self_id"`
	PostType    string          `json:"post_type"`
	MessageType string          `json:"message_type"`
	SubType     string          `json:"sub_type"`
	UserID      json.RawMessage `json:"user_id"`
	GroupID     json.RawMessage `json:"group_id"`
	MessageID   json.RawMessage `json:"message_id"`
	Message     json.RawMessage `json:"message"`
	Raw         map[string]any  `json:"-"`
}

func (e *Event) UnmarshalJSON(data []byte) error {
	type alias Event
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	eventMap := make(map[string]any)
	if err := json.Unmarshal(data, &eventMap); err != nil {
		return err
	}
	*e = Event(decoded)
	e.Raw = eventMap
	if len(e.Message) > 0 {
		e.Message, _ = json.Marshal(ParseMessage(e.Message))
		e.Raw["message"] = ParseMessage(e.Message)
	}
	return nil
}

func (e Event) PlainText() string {
	segments := ParseMessage(e.Message)
	var result strings.Builder
	for _, segment := range segments {
		if segment.Type == "text" {
			if value, ok := segment.Data["text"].(string); ok {
				result.WriteString(value)
			}
		}
	}
	return result.String()
}
