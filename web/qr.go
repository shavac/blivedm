package web

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/shavac/blivedm"
	"github.com/skip2/go-qrcode"
)

func RenderLoginQRCode(w io.Writer, value string) error {
	code, err := qrcode.New(value, qrcode.Medium)
	if err != nil {
		return fmt.Errorf("encode QR code: %w", err)
	}
	bitmap := code.Bitmap()
	for y := 0; y < len(bitmap); y += 2 {
		for x := range bitmap[y] {
			top := bitmap[y][x]
			bottom := y+1 < len(bitmap) && bitmap[y+1][x]
			switch {
			case top && bottom:
				fmt.Fprint(w, "█")
			case top:
				fmt.Fprint(w, "▀")
			case bottom:
				fmt.Fprint(w, "▄")
			default:
				fmt.Fprint(w, " ")
			}
		}
		fmt.Fprintln(w)
	}
	return nil
}

func CompleteLogin(ctx context.Context, session *blivedm.Session, out io.Writer) error {
	client := blivedm.NewClient(session)
	nav, err := client.Nav(ctx)
	if err != nil {
		return fmt.Errorf("verify session: %w", err)
	}
	if !nav.IsLogin || nav.MID == 0 {
		return fmt.Errorf("bilibili rejected the session; the previous session was kept")
	}
	session.DedeUserID = nav.MID
	if session.BUVID3 == "" {
		buvid3, err := client.DeviceID(ctx)
		if err != nil {
			return err
		}
		session.BUVID3 = buvid3
	}
	session.SavedAt = time.Now().Format(time.RFC3339)
	if err := blivedm.SaveSession(session); err != nil {
		return err
	}
	if session.LoggedIn() {
		fmt.Fprintf(out, "已登录：%s (mid %d)，可以发送评论。\n", nav.Uname, nav.MID)
	} else {
		fmt.Fprintf(out, "已登录：%s (mid %d)，缺少 bili_jct，无法发送评论。\n", nav.Uname, nav.MID)
	}
	return nil
}
