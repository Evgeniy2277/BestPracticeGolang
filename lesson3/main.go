package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)


type CrawlResult struct {
	Err   error
	Title string
	Url   string
}

type Page interface {
	GetTitle() string
	GetLinks() []string
}

type page struct {
	doc *goquery.Document
}

func NewPage(raw io.Reader) (Page, error) {
	doc, err := goquery.NewDocumentFromReader(raw)
	if err != nil {
		return nil, err
	}
	return &page{doc: doc}, nil
}

func (p *page) GetTitle() string {
	return p.doc.Find("title").First().Text()
}

func (p *page) GetLinks() []string {
	var urls []string
	p.doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		url, ok := s.Attr("href")
		if ok {
			urls = append(urls, url)
		}
	})
	return urls
}

type Requester interface {
	Get(ctx context.Context, url string) (Page, error)
}

type requester struct {
	timeout time.Duration
}

func NewRequester(timeout time.Duration) requester {
	return requester{timeout: timeout}
}

func (r requester) Get(ctx context.Context, url string) (Page, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:
		cl := &http.Client{
			Timeout: r.timeout,
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		body, err := cl.Do(req)
		if err != nil {
			return nil, err
		}
		defer body.Body.Close()
		page, err := NewPage(body.Body)
		if err != nil {
			return nil, err
		}
		return page, nil
	}
	
}

//Crawler - интерфейс (контракт) краулера
type Crawler interface {
	Scan(ctx context.Context, url string, depth int)
	ChanResult() <-chan CrawlResult
}

type crawler struct {
	r       Requester
	res     chan CrawlResult
	visited map[string]struct{}
	mu      sync.RWMutex
	chUpDepth chan bool
	logger *zap.Logger

}

func NewCrawler(r Requester, loger *zap.Logger) *crawler {
	return &crawler{
		r:       r,
		res:     make(chan CrawlResult),
		visited: make(map[string]struct{}),
		mu:      sync.RWMutex{},
		chUpDepth: make(chan bool, 1),
		logger:  loger,

	}
}

func (c *crawler) Scan(ctx context.Context, url string, depth int) {


	if depth <= 0 { //Проверяем то, что есть запас по глубине
		return
	}
	c.mu.RLock()
	_, ok := c.visited[url] //Проверяем, что мы ещё не смотрели эту страницу
	c.mu.RUnlock()
	if ok {
		return
	}
	
	select {

	case <-c.chUpDepth:
		depth += 2
		log.Default().Println("dept+ = ", depth)

	case <-ctx.Done(): //Если контекст завершен - прекращаем выполнение
		return
		
	case <- time.After(1 * time.Microsecond):
		var err error
		c.res <- CrawlResult{Err: fmt.Errorf("timeout (*crawler).Scan %t", err)}
		return

	default:
		page, err := c.r.Get(ctx, url) //Запрашиваем страницу через Requester
		if err != nil {
			c.res <- CrawlResult{Err: err} //Записываем ошибку в канал
			return
		}
		c.mu.Lock()
		c.visited[url] = struct{}{} //Помечаем страницу просмотренной
		c.mu.Unlock()
		c.res <- CrawlResult{ //Отправляем результаты в канал
			Title: page.GetTitle(),
			Url:   url,
		}		

		for _, link := range page.GetLinks() {
			if depth == 1 {
				c.logger.Panic("dept == 1 PANIC message")
			}
			go c.Scan(ctx, link, depth-1) //На все полученные ссылки запускаем новую рутину сборки
		}
	}
}

func (c *crawler) ChanResult() <-chan CrawlResult {
	return c.res
}

//Config - структура для конфигурации
type Config struct {
	MaxDepth   int
	MaxResults int
	MaxErrors  int
	Url        string
	Timeout    int //in seconds
}

func main() {

    logger, _ := zap.NewDevelopment()
	logger.Debug("This is a DEBUG message")
	// logger.Debug("This is a DEBUG message")
	logger.Info("This is an INFO message")
	logger.Info("This is an INFO message with fields", zap.String("region", "us-west"), zap.Int("id", 2))
	// logger.Warn("This is a WARN message")
	// logger.Error("This is an ERROR message")
	// logger.Fatal("This is a FATAL message")   // would exit if uncommented
	// logger.DPanic("This is a DPANIC message") // would exit if uncommented
	// logger.Panic("This is a PANIC message")    // would exit if uncommented
	rawJSONConfig := []byte(`{
		"level": "info",
		"encoding": "console",
		"outputPaths": ["stdout", "/tmp/logs"],
		"errorOutputPaths": ["/tmp/errorlogs"],
		"initialFields": {"initFieldKey": "fieldValue"},
		"encoderConfig": {
		"messageKey": "message",
		"levelKey": "level",
		"nameKey": "logger",
		"timeKey": "time",
		"callerKey": "logger",
		"stacktraceKey": "stacktrace",
		"callstackKey": "callstack",
		"errorKey": "error",
		"timeEncoder": "iso8601",
		"fileKey": "file",
		"levelEncoder": "capitalColor",
		"durationEncoder": "second",
		"callerEncoder": "full",
		"nameEncoder": "full",
		"sampling": {
			"initial": "3",
			"thereafter": "10"
		}
		}
	}`)

	config := zap.Config{}
	if err := json.Unmarshal(rawJSONConfig, &config); err != nil {
		panic(err)
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	cfg := Config{
		MaxDepth:   3,
		MaxResults: 1000,
		MaxErrors:  500,
		Url:        "https://telegram.org",
		Timeout:    100,	
	}
	//var cr Crawler ругался тест
	//var r Requester

	r := NewRequester(time.Duration(cfg.Timeout) * time.Second)
	cr := NewCrawler(r, logger)
	ctx, cancel := context.WithCancel(context.Background())
	// ctx, cancel := context.WithTimeout(ctx, time.Second * 300)
	// _ = cancel1
	go cr.Scan(ctx, cfg.Url, cfg.MaxDepth) //Запускаем краулер в отдельной рутине
	go processResult(ctx, cancel, cr, cfg) //Обрабатываем результаты в отдельной рутине

	sigCh := make(chan os.Signal, 1)        //Создаем канал для приема сигналов, ругался тест добавил 1
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGUSR1) //Подписываемся на сигнал SIGINT, SIGUSR1
 
    
	for {
		select {
			case <-ctx.Done(): 
    			    return					
		case sig := <-sigCh:
				if sig == syscall.SIGINT  {
				    log.Println("Cигнал SigInt - завершаем контекст")
				    cancel()
				} else if sig == syscall.SIGUSR1 {
					cr.chUpDepth <- true
				}
		}						
	}
}

func processResult(ctx context.Context, cancel func(), cr Crawler, cfg Config) {
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
