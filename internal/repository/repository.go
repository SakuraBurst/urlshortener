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

func (m MapBd) Read(ctx context.Context, id string) *URLTransfer {
	urlChan := make(chan *URLTransfer)
	clh := closedHelper{closed: make(chan bool)}
	go getFromBd(urlChan, clh, id)
	select {
	case urlTransfer := <-urlChan:
		return urlTransfer
	case <-ctx.Done():
		log.Println("context canceled with", ctx.Err())
		close(clh.closed)
		close(urlChan)
		return &URLTransfer{
			UnShorterURL: nil,
			Err:          ctx.Err(),
		}
	}
}

func (m MapBd) Write(ctx context.Context, u *url.URL) *ResultTransfer {
	resultChan := make(chan *ResultTransfer)
	clh := closedHelper{closed: make(chan bool)}
	go writeToBd(resultChan, clh, u)
	select {
	case res := <-resultChan:
		return res
	case <-ctx.Done():
		log.Println("context canceled with", ctx.Err())
		close(clh.closed)
		close(resultChan)
		return &ResultTransfer{Err: ctx.Err()}
	}

}

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

type closedHelper struct {
	closed chan bool
}

func (ch *closedHelper) isClosed() bool {
	select {
	case <-ch.closed:
		return true
	default:
		return false
	}
}

func getFromBd(urlChan chan<- *URLTransfer, clh closedHelper, id string) {
	var err error = nil
	unShorterURL, ok := bd[id]
	if !ok {
		err = ErrorNoSuchURL
	}
	if clh.isClosed() {
		return
	}
	urlChan <- &URLTransfer{
		UnShorterURL: unShorterURL,
		Err:          err,
	}
}

func writeToBd(resultChan chan<- *ResultTransfer, clh closedHelper, unShortenURL *url.URL) {
	h := sha1.New()
	_, err := io.WriteString(h, unShortenURL.String())
	if err != nil {
		if !clh.isClosed() {
			resultChan <- &ResultTransfer{Err: err}
		}
		return
	}
	result := fmt.Sprintf("%x", h.Sum(nil))[:5]
	bd[result] = unShortenURL
	if clh.isClosed() {
		return
	}
	resultChan <- &ResultTransfer{ID: result}
}
