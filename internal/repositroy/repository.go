package repositroy

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
)

var bd = map[string]*url.URL{}

type URLTransfer struct {
	UnShorterURL *url.URL
	Err          error
}

type ResultTransfer struct {
	Id  string
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

func WriteToBd(ctx context.Context, unShorterURL *url.URL) *ResultTransfer {
	resultChan := make(chan *ResultTransfer)
	clh := closedHelper{closed: make(chan bool)}
	go writeToBd(resultChan, clh, unShorterURL)
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

func GetFromBd(ctx context.Context, id string) *URLTransfer {
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

func getFromBd(urlChan chan<- *URLTransfer, clh closedHelper, id string) {
	var err error = nil
	unShorterUrl, ok := bd[id]
	if !ok {
		err = errors.New("there is no such url")
	}
	if clh.isClosed() {
		return
	}
	urlChan <- &URLTransfer{
		UnShorterURL: unShorterUrl,
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
	resultChan <- &ResultTransfer{Id: result}
}
