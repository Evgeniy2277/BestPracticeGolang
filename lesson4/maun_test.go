package main

import (
	"testing"
	"time"
	"go.uber.org/zap"
	"github.com/stretchr/testify/assert"
	"sync"
)

func TestNewCrawler(t *testing.T) {
	tr := NewRequester(3 * time.Second)
	tlogger, _ := zap.NewDevelopment()
	var twg sync.WaitGroup
    tNewCra := NewCrawler(tr, tlogger, &twg)
	assert.NotNil(t, NewCrawler(tr, tlogger, &twg), tNewCra)
}
