package crawl

import (
	"context"
	"lesson5/request"
	"sync"
)

type CrawlResult struct {
	Err   error
	Title string
	Url   string
}

//Crawler - интерфейс (контракт) краулера
type Crawler interface {
	Scan(ctx context.Context, url string, depth int)
	ChanResult() <-chan CrawlResult
}

type Crawl struct {
	r       request.Requester
	res     chan CrawlResult
	visited map[string]struct{}
	mu      sync.RWMutex
}

func NewCrawler(r request.Requester) *Crawl {
	return &Crawl{
		r:       r,
		res:     make(chan CrawlResult),
		visited: make(map[string]struct{}),
		mu:      sync.RWMutex{},
	}
}

func (c *Crawl) Scan(ctx context.Context, url string, depth int) {

	// if sig2 != 0 {
	// 	depth = depth + sig2 // добавляем при приходе SIGUSER1 глубину
	// 	sig2 = 0
	// }
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
	case <-ctx.Done(): // Если контекст завершен - прекращаем выполнение
		return
	default:
		// Запрашиваем страницу через Requester
		page, err := c.r.Get(ctx, url)
		if err != nil {
			c.res <- CrawlResult{Err: err} // Записываем ошибку в канал
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
			go c.Scan(ctx, link, depth-1) //На все полученные ссылки запускаем новую рутину сборки
		}
	}
}

func (c *Crawl) ChanResult() <-chan CrawlResult {
	return c.res
}
