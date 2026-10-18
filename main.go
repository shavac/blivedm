package main

import (
	"context"
	"flag"
	"fmt"

	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shavac/blivedm"
	"github.com/shavac/robao/web"
)

func main() {
	roomId := flag.Int("r", 0, "Room ID")
	flag.Parse()
	// 填入真实房间号或短号，内部会自动解析
	if *roomId <= 0 {
		log.Fatalln("room id < 0")
		return
	}
	if err := blivedm.ClearSession(); err != nil {
		log.Fatalf("%v", err)
	}

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
	if err := web.RenderLoginQRCode(os.Stdout, login.URL); err != nil {
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
			return
		case blivedm.QRLoginSuccess:
			status.Session.BUVID3 = buvid3
			web.CompleteLogin(pollCtx, status.Session, os.Stdout)
			return
		}
		state = status.State
		select {
		case <-pollCtx.Done():
			log.Fatalf("%v", pollCtx.Err())
		case <-ticker.C:
		}
	}

}
