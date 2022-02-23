package main

import (
	"context"
	"io"
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
	wg sync.WaitGroup
}

func NewCrawler(r Requester, loger *zap.Logger, wwg *sync.WaitGroup) *crawler {
	return &crawler{
		r:       r,
		res:     make(chan CrawlResult),
		visited: make(map[string]struct{}),
		mu:      sync.RWMutex{},
		chUpDepth: make(chan bool, 1),
		logger:  loger,
		wg: *wwg,
	}
}

func (c *crawler) Scan(ctx context.Context, url string, depth int) {
	defer c.wg.Done()
	// defer fmt.Println("wg = wd - 1")
	
	if depth <= 0 { 
		c.logger.Info("depth <= 0 in Scan", zap.Int("", depth))
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
		c.logger.Debug("dept+ = ", zap.Int("depth", depth))

	case <-ctx.Done(): //Если контекст завершен - прекращаем выполнение
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
			c.wg.Add(1)
			go c.Scan(ctx, link, depth-1) 
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
	Timeout    int 
	Time2      int
	
}

func main() {
	var wg sync.WaitGroup

	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	logger.Info("This is an INFO message RUN Logger ZAP")
		
	cfg := Config{
		MaxDepth:   3,
		MaxResults: 4,
		MaxErrors:  4,
		Url:        "https://telegram.org",
		Timeout:    3,
		Time2:      30,	
	}
	r := NewRequester(time.Duration(cfg.Timeout) * time.Second)
	cr := NewCrawler(r, logger, &wg)
	ctx, cancel := context.WithCancel(context.Background())  //lint:ignore SA4006 we love not used varible!
	ctx, cancel = context.WithTimeout(ctx, time.Duration(cfg.Time2) * time.Second)	
	wg.Add(1)	
	go cr.Scan(ctx, cfg.Url, cfg.MaxDepth) //Запускаем краулер в отдельной рутине
	go processResult(ctx, cancel, cr, cfg, logger) //Обрабатываем результаты в отдельной рутине

	sigCh := make(chan os.Signal, 1)       
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGUSR1) //Подписываемся на сигнал SIGINT, SIGUSR1
 
 label:   
	for {
		select {
			case <-ctx.Done(): 		
			logger.Info("context Done")
			        break label				
		case sig := <-sigCh:
				if sig == syscall.SIGINT  {
				    logger.Info("sycsll.SYGINT")
				    cancel()
				} else if sig == syscall.SIGUSR1 {
					logger.Info("syscall.SIGUSR1")
					cr.chUpDepth <- true
				}
		}			
    				
	}
	logger.Debug("Wait wg.Wait ")
	wg.Wait()	
}

func processResult(ctx context.Context, cancel func(), cr Crawler, cfg Config, logger *zap.Logger) {
	var maxResult, maxErrors = cfg.MaxResults, cfg.MaxErrors
	
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cr.ChanResult():
			if msg.Err != nil {
				maxErrors--	
				logger.Debug(msg.Err.Error())	
				if maxErrors <= 0 {
				logger.Warn("maxErrors <=0 ")						
					cancel()
					return
				}
			} else {
				maxResult--
				logger.Info("crawler result: ", zap.String(msg.Url, msg.Title), )
				if maxResult <= 0 {
					logger.Error("maxresult <=0")
					cancel()
					return
				}
			}
		}
	}


}
