package repository

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
)

type MapBd map[string]*url.URL

var ErrorNoSuchURL = errors.New("there is no such url")

var bd = MapBd{}

type URLShortenerRepository interface {
	Read(context.Context, string) *URLTransfer
	Write(context.Context, *url.URL) *ResultTransfer
}

type URLTransfer struct {
	UnShorterURL *url.URL
	Err          error
}

type ResultTransfer struct {
	ID  string
	Err error
}

func (m MapBd) Read(ctx context.Context, id string) *URLTransfer {
	urlChan := make(chan *URLTransfer)
	go getFromBd(ctx, urlChan, id)
	select {
	case urlTransfer := <-urlChan:
		return urlTransfer
	case <-ctx.Done():
		log.Println("context canceled with ", ctx.Err())
		close(urlChan)
		return &URLTransfer{
			UnShorterURL: nil,
			Err:          ctx.Err(),
		}
	}
}

func (m MapBd) Write(ctx context.Context, u *url.URL) *ResultTransfer {
	resultChan := make(chan *ResultTransfer)
	go writeToBd(ctx, resultChan, u)
	select {
	case res := <-resultChan:
		return res
	case <-ctx.Done():
		log.Println("context canceled with ", ctx.Err())
		close(resultChan)
		return &ResultTransfer{Err: ctx.Err()}
	}

}

func getFromBd(ctx context.Context, urlChan chan<- *URLTransfer, id string) {
	var err error = nil
	unShorterURL, ok := bd[id]
	if !ok {
		err = ErrorNoSuchURL
	}
	if ctx.Err() != nil {
		return
	}
	urlChan <- &URLTransfer{
		UnShorterURL: unShorterURL,
		Err:          err,
	}
}

func writeToBd(ctx context.Context, resultChan chan<- *ResultTransfer, unShortenURL *url.URL) {
	h := sha1.New()
	_, err := io.WriteString(h, unShortenURL.String())
	if err != nil {
		if ctx.Err() == nil {
			resultChan <- &ResultTransfer{Err: err}
		}
		return
	}
	result := fmt.Sprintf("%x", h.Sum(nil))[:5]
	bd[result] = unShortenURL
	if ctx.Err() != nil {
		return
	}
	resultChan <- &ResultTransfer{ID: result}
}
