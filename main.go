package main

import (
	"flag"

	"log"

	"github.com/shavac/robao/web"
)

func main() {
	roomId := flag.Int64("r", 0, "Room ID")
	flag.Parse()
	// 填入真实房间号或短号，内部会自动解析
	if *roomId <= 0 {
		log.Fatalln("room id < 0")
		return
	}

	client, err := web.CreateBliveClient()
	if err != nil {
		log.Fatalf("login: %v", err)
	}
	_ = client
	select {}
}
