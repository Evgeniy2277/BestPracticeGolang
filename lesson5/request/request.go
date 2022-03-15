package request

import (
	"context"
	// "fmt"
	// "lesson5/crawl"
	"lesson5/page"
	// "log"
	"net/http"
	// "os"
	// "os/signal"
	// "sync"
	// "syscall"
	"time"
)

type Requester interface {
	Get(ctx context.Context, url string) (page.Pageer, error) ///
}

type Request struct {
	timeout time.Duration
}

func NewRequester(timeout time.Duration) Request {
	return Request{timeout: timeout}
}

func (r Request) Get(ctx context.Context, url string) (page.Pageer, error) {               ///
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
		Page, err := page.NewPage(body.Body)
		if err != nil {
			return nil, err
		}
		return Page, nil
	}

}