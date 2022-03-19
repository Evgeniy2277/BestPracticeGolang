package page

import (
    "io"		
    "github.com/PuerkitoBio/goquery"
)

type Pageer interface {
	GetTitle() string
	GetLinks() []string
}

type Page struct {
	doc *goquery.Document
}

func NewPage(raw io.Reader) (Pageer, error) {                    
	doc, err := goquery.NewDocumentFromReader(raw)
	if err != nil {
		return nil, err
	}
	return &Page{doc: doc}, nil
}

func (p *Page) GetTitle() string {                                 
	return p.doc.Find("title").First().Text()
}

func (p *Page) GetLinks() []string {                              
	var urls []string
	p.doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		url, ok := s.Attr("href")
		if ok {
			urls = append(urls, url)
		}
	})
	return urls
}
