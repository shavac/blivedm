package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shavac/blivedm"
	"github.com/skip2/go-qrcode"
)

func CreateBliveClient() (*blivedm.Client, error) {
	session, err := blivedm.LoadSession()
	if err != nil {
		log.Println("warning: load session:", err)
	}
	if session != nil && session.HasSession() && session.LoggedIn() {
		client := blivedm.NewClient(session)
		if _, err = client.Nav(context.Background()); err == nil {
			return client, nil
		} else {
			blivedm.ClearSession()
		}
	}
	return QRLogin()
}

func QRLogin() (*blivedm.Client, error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := blivedm.NewClient(nil)
	buvid3, err := client.DeviceID(ctx)
	if err != nil {
		log.Fatalf("%v", err)
	}
	login, err := client.NewQRLogin(ctx)
	if err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Println("请使用哔哩哔哩客户端扫描二维码登录：")
	if err := RenderLoginQRCode(os.Stdout, login.URL); err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Println("扫描后请在手机上确认。")

	pollCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	state := blivedm.QRLoginWaiting
	for {
		status, err := client.PollQRLogin(pollCtx, login.Key)
		if err != nil {
			if pollCtx.Err() != nil {
				log.Fatalf("%v", pollCtx.Err())
			}
		}
		switch status.State {
		case blivedm.QRLoginWaiting:
		case blivedm.QRLoginScanned:
			if state != status.State {
				fmt.Fprintln(os.Stdout, "已扫码，请在手机上确认。")
			}
		case blivedm.QRLoginExpired:
			log.Println("二维码已过期，请重新运行 `bili login`")
			return nil, errors.New("qr code outdated")
		case blivedm.QRLoginSuccess:
			status.Session.BUVID3 = buvid3
			CompleteLogin(pollCtx, status.Session, os.Stdout)
			return client, nil
		}
		state = status.State
		select {
		case <-pollCtx.Done():
			log.Fatalf("%v", pollCtx.Err())
		case <-ticker.C:
		}
	}
}

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
