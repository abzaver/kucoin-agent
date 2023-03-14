package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Kucoin/kucoin-go-sdk"
)

func main() {
	// API key version 2.0
	s := kucoin.NewApiService(
		kucoin.ApiBaseURIOption("https://api.kucoin.com"),
	)
	serverTime(s)

	// Setup a channel to listen for interrupt signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Create a context that will be canceled on SIGINT
	ctx, cancel := context.WithCancel(context.Background())

	go publicWebsocket(s, ctx)

	// Wait for an interrupt
	sig := <-sigs
	log.Printf("Received signal: %s. Shutting down gracefully...", sig)
	cancel()

	// Give some time for goroutines to finish
	time.Sleep(2 * time.Second)
}

func serverTime(s *kucoin.ApiService) {
	rsp, err := s.ServerTime()
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return
	}

	var ts int64
	if err := rsp.ReadData(&ts); err != nil {
		log.Printf("Error: %s", err.Error())
		return
	}
	log.Printf("The server time: %d", ts)
}

func publicWebsocket(s *kucoin.ApiService, ctx context.Context) {
	rsp, err := s.WebSocketPublicToken()
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return
	}

	tk := &kucoin.WebSocketTokenModel{}
	if err := rsp.ReadData(tk); err != nil {
		log.Printf("Error: %s", err.Error())
		return
	}

	c := s.NewWebSocketClient(tk)

	mc, ec, err := c.Connect()
	if err != nil {
		log.Printf("Error: %s", err.Error())
		return
	}

	ch1 := kucoin.NewSubscribeMessage("/market/ticker:KCS-BTC", false)
	//ch1 := kucoin.NewSubscribeMessage("/market/ticker:ETH-BTC", false)
	//ch1 := kucoin.NewSubscribeMessage("/market/ticker:all", false)

	if err := c.Subscribe(ch1); err != nil {
		log.Printf("Error: %s", err.Error())
		return
	}

	for {
		select {
		case err := <-ec:
			c.Stop() // Stop subscribing the WebSocket feed
			log.Printf("Error: %s", err.Error())
			return
		case msg := <-mc:
			log.Printf("Received: %s", kucoin.ToJsonString(msg))
		case <-ctx.Done():
			log.Println("Exit subscription")
			c.Stop() // Stop subscribing the WebSocket feed
			return
		}
	}
}
