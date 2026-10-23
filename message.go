package blivedm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shavac/blivedm/proto"
	googleproto "google.golang.org/protobuf/proto"
)

type Danmu struct {
	RoomID     int64
	UID        int64
	Uname      string
	Content    string
	MedalName  string
	MedalLevel int64
	GuardLevel int64
	Timestamp  time.Time
}

func ParseDanmu(msg map[string]any) (*Danmu, error) {
	dm := &Danmu{}
	if msg == nil {
		return nil, fmt.Errorf("invalid message")
	}
	info, ok := msg["info"].([]any)
	if !ok || len(info) < 3 {
		return nil, fmt.Errorf("invalid message info")
	}
	if len(info) > 1 {
		dm.Content = anyToString(info[1])
	}
	if sender, ok := info[2].([]any); ok && len(sender) >= 2 {
		dm.UID = parseInt64(sender[0])
		dm.Uname = anyToString(sender[1])
		if len(sender) > 3 {
			dm.MedalLevel = parseInt64(sender[3])
		}
	}
	if len(info) > 3 {
		if m, ok := info[3].([]any); ok && len(m) >= 2 {
			dm.MedalName = anyToString(m[1])
			dm.MedalLevel = parseInt64(m[0])
			if len(m) > 10 {
				dm.GuardLevel = parseInt64(m[10])
			}
		}
	}
	//dm.Timestamp = time.Unix(parseInt64(msg["timestamp"]), 0)
	dm.Timestamp = time.Now()
	return dm, nil
}

func anyToString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case float64:
		if s == float64(int64(s)) {
			return strconv.FormatInt(int64(s), 10)
		}
		return strconv.FormatFloat(s, 'f', -1, 64)
	case json.Number:
		return s.String()
	case nil:
		return ""
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func parseInt64(v any) int64 {
	s := anyToString(v)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

type Gift struct {
	RoomID     int64
	UID        int64
	Uname      string
	GiftName   string
	GiftCount  int64
	Price      string
	GuardLevel int64
	Timestamp  time.Time
}

func ParseGift(msg map[string]any, combo bool) (*Gift, error) {
	g := &Gift{}
	d := payloadOf(msg)
	g.UID = int64(num(d["uid"]))
	g.Uname = str(d, "uname")
	g.GiftName = str(d, "giftName")
	if g.GiftName == "" {
		g.GiftName = str(d, "gift_name")
	}
	g.GiftCount = int64(num(d["num"]))
	if combo {
		g.GiftCount = int64(num(d["combo_num"]))
	}
	coin := int64(num(d["total_coin"]))
	if combo {
		coin = int64(num(d["combo_total_coin"]))
	}
	ctype := str(d, "coin_type")
	if ctype == "silver" {
		g.Price = fmt.Sprintf("%d银", coin/100)
	} else {
		g.Price = fmt.Sprintf("¥%.1f", float64(coin)/1000)
	}
	guard := int64(num(d["guard_level"]))
	if guard > 0 {
		g.GuardLevel = guard
	}
	g.Timestamp = time.Now()
	return g, nil

}

// payloadOf returns the object that renderers read. It prefers the nested
// data object. Older wire formats keep the fields on the top level.
func payloadOf(body map[string]any) map[string]any {
	if d, ok := body["data"].(map[string]any); ok && len(d) > 0 {
		return d
	}
	return body
}

func num(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	}
	return 0
}

func str(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	if v, ok := m[key]; ok {
		return anyToString(v)
	}
	// Dotted path lookup, for example user_info.uname.
	if strings.Contains(key, ".") {
		parts := strings.SplitN(key, ".", 2)
		if next, ok := m[parts[0]].(map[string]any); ok {
			return str(next, parts[1])
		}
	}
	return ""
}

type Guard struct {
	RoomID     int64
	UID        int64
	Uname      string
	GuardType  int64
	GuardLevel int64
	Timestamp  time.Time
}

func ParseGuard(msg map[string]any) (*Guard, error) {
	g := &Guard{}
	d := payloadOf(msg)
	g.UID = int64(num(d["uid"]))
	g.Uname = str(d, "username")
	g.GuardType = int64(num(d["guard_type"]))
	g.GuardLevel = int64(num(d["guard_level"]))
	g.Timestamp = time.Now()
	return g, nil
}

type SuperChat struct {
	RoomID    int64
	UID       int64
	Uname     string
	Message   string
	Price     int64
	StartTime time.Time
	EndTime   time.Time
	Color     string
	Timestamp time.Time
}

func ParseSuperChat(msg map[string]any) (*SuperChat, error) {
	sc := &SuperChat{}
	d := payloadOf(msg)
	sc.UID = int64(num(d["uid"]))
	sc.Uname = str(d, "user_info.uname")
	sc.Message = str(d, "message")
	sc.Price = int64(num(d["price"]))
	startTime := int64(num(d["start_time"]))
	endTime := int64(num(d["end_time"]))
	if startTime > 0 {
		sc.StartTime = time.Unix(startTime, 0)
	}
	if endTime > 0 {
		sc.EndTime = time.Unix(endTime, 0)
	}
	sc.Color = str(d, "background_bottom_color")
	sc.Timestamp = time.Now()
	return sc, nil
}

type ActType uint32

const (
	ActTypeEnterRoom ActType = 1
	ActTypeFollow    ActType = 2
	ActTypeShare     ActType = 3
	ActTypeLike      ActType = 4
)

type Interact struct {
	RoomID    uint32
	UID       uint64
	Uname     string
	Action    ActType
	GuardType uint32
	Timestamp time.Time
}

func ParseInteractV2(msg map[string]any) (*Interact, error) {
	d := payloadOf(msg)
	rawBytes, err := base64.StdEncoding.DecodeString(d["pb"].(string))

	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}
	var iword proto.EventInteractWord
	err = googleproto.Unmarshal(rawBytes, &iword)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal protobuf: %v", err)
	}
	ita := &Interact{
		RoomID:    iword.GetRoomId(),
		UID:       iword.GetUid(),
		Uname:     iword.GetUsername(),
		Action:    ActType(iword.GetMsgType()),
		GuardType: iword.GetGuardType(),
		Timestamp: time.Unix(int64(iword.GetTimestamp()), 0),
	}
	return ita, nil
}
