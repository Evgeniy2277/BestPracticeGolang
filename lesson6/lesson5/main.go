package main

import (
	"context"
	"fmt"
	"lesson5/crawl"
	"lesson5/request"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var sig2 int

//Config - структура для конфигурации
type Config struct {
	MaxDepth   int
	MaxResults int
	MaxErrors  int
	Url        string
	Timeout    int //in seconds
	Timeout2   int // seconds for newPage() GetLincks() GetTitle()
}

func main() {

	cfg := Config{
		MaxDepth:   3,
		MaxResults: 1000,
		MaxErrors:  500,
		Url:        "https://telegram.org",
		Timeout:    100,
		Timeout2:   3,
	}

	r := request.NewRequester(time.Duration(cfg.Timeout) * time.Second)
	cr := crawl.NewCrawler(r)
	sig2 = 0
	ctx, cancel := context.WithCancel(context.Background())
	go cr.Scan(ctx, cfg.Url, cfg.MaxDepth) //Запускаем краулер в отдельной рутине
	go processResult(ctx, cancel, cr, cfg) //Обрабатываем результаты в отдельной рутине

	sigCh := make(chan os.Signal, 1)                      //Создаем канал для приема сигналов, ругался тест добавил 1
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGUSR1) //Подписываемся на сигнал SIGINT, SIGUSR1

	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-sigCh:
			if sig == syscall.SIGINT {
				fmt.Println("Если пришёл сигнал SigInt - завершаем контекст")
				cancel()
			} else if sig == syscall.SIGUSR1 {
				sig2 += 2
				fmt.Println("Если пришёл сигнал SigUSR1 - прибавляем к depth 2")
			}
		}
	}
}

func processResult(ctx context.Context, cancel func(), cr crawl.Crawler, cfg Config) {
	var maxResult, maxErrors = cfg.MaxResults, cfg.MaxErrors
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cr.ChanResult():
			if msg.Err != nil {
				maxErrors--
				log.Printf("crawler result return err: %s\n", msg.Err.Error())
				if maxErrors <= 0 {
					cancel()
					return
				}
			} else {
				maxResult--
				log.Printf("crawler result: [url: %s] Title: %s\n", msg.Url, msg.Title)
				if maxResult <= 0 {
					cancel()
					return
				}
			}
		}
	}
}
