package main

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"sync"
	"testing"
	"time"
)

func TestNewCrawler(t *testing.T) {
	tr := NewRequester(3 * time.Second)
	tlogger, _ := zap.NewDevelopment()
	var twg sync.WaitGroup
	tNewCra := NewCrawler(tr, tlogger, &twg)
	assert.NotNil(t, NewCrawler(tr, tlogger, &twg), tNewCra)
}
