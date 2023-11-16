package main

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func upgradeToWebSocket() {
	var c *websocket.Conn
	var err error

	for {
		c, _, err = websocket.DefaultDialer.Dial("ws://localhost:8080/api/v2/cluster?follow=true", nil)
		if err != nil {
			log.Println("dial:", err)
			time.Sleep(time.Second * 5) // wait for 5 seconds before retry
			continue
		}

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}
			log.Printf("recv: %s", message)
		}

		c.Close()
	}
}

func main() {
	upgradeToWebSocket()
	// m := make(chan interface{}, 200)
	// cli := client.New("http://localhost:8080", "")

	// var cls cluster.Cluster
	// if err := cli.Get("4", &cls); err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("%v\n", cls)
	// fmt.Println("+++++++++++++++++++++++++++++++++++++")

	// var clss []cluster.Cluster

	// if err := cli.List(&clss); err != nil {
	// 	log.Fatal(err)
	// }
	// for _, cls := range clss {
	// 	log.Printf("%v\n", cls)
	// }
	// fmt.Println("=====================================")

	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// if err := cli.Watch(ctx, m, &clss); err != nil {
	// 	log.Fatal(err)
	// }
	// go func() {
	// for {
	// 	select {
	// 	case msg := <-m:
	// 		log.Printf("recv: %+v\n", msg)
	// 	}
	// }
	// }()
	// time.Sleep(time.Second * 5)
	// fmt.Println("close")

	// fmt.Println("closed")
	// time.Sleep(time.Second * 5)
	// fmt.Println("exit")
	// cancel()
}
