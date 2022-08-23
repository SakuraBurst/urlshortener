package repository

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync"
)

type MapBd struct {
	sync.Map
}

var ErrNoSuchURL = errors.New("there is no such url")
var ErrUnexpectedTypeInMap = errors.New("unexpected type in map")

type URLShortenerRepository interface {
	ReadFromBd(context.Context, string) *URLTransfer
	WriteToBd(context.Context, *url.URL) *ResultTransfer
	SetKeyValue(string, *url.URL)
}

type URLTransfer struct {
	UnShorterURL *url.URL
	Err          error
}

type ResultTransfer struct {
	ID  string
	Err error
}

func (m *MapBd) SetKeyValue(key string, value *url.URL) {
	m.Store(key, value)
}

func (m *MapBd) ReadFromBd(ctx context.Context, id string) *URLTransfer {
	urlChan := make(chan *URLTransfer)
	go m.getFromBd(ctx, urlChan, id)
	select {
	case urlTransfer := <-urlChan:
		return urlTransfer
	case <-ctx.Done():
		close(urlChan)
		return &URLTransfer{
			UnShorterURL: nil,
			Err:          ctx.Err(),
		}
	}
}

func (m *MapBd) WriteToBd(ctx context.Context, u *url.URL) *ResultTransfer {
	resultChan := make(chan *ResultTransfer)
	go m.writeToBd(ctx, resultChan, u)
	select {
	case res := <-resultChan:
		return res
	case <-ctx.Done():
		close(resultChan)
		return &ResultTransfer{Err: ctx.Err()}
	}

}

func (m *MapBd) getFromBd(ctx context.Context, urlChan chan<- *URLTransfer, id string) {
	var err error = nil
	untypedURL, ok := m.Load(id)
	if !ok {
		if ctx.Err() == nil {
			urlChan <- &URLTransfer{
				UnShorterURL: nil,
				Err:          ErrNoSuchURL,
			}
		}
		return
	}
	var unShorterURL *url.URL
	switch v := untypedURL.(type) {
	case *url.URL:
		unShorterURL = v
	default:
		if ctx.Err() == nil {
			urlChan <- &URLTransfer{
				UnShorterURL: nil,
				Err:          ErrUnexpectedTypeInMap,
			}
		}
		return
	}
	if ctx.Err() == nil {
		urlChan <- &URLTransfer{
			UnShorterURL: unShorterURL,
			Err:          err,
		}
	}
}

func (m *MapBd) writeToBd(ctx context.Context, resultChan chan<- *ResultTransfer, unShortenURL *url.URL) {
	h := sha1.New()
	_, err := io.WriteString(h, unShortenURL.String())
	if err != nil {
		if ctx.Err() == nil {
			resultChan <- &ResultTransfer{Err: err}
		}
		return
	}
	result := fmt.Sprintf("%x", h.Sum(nil))[:5]
	m.Store(result, unShortenURL)
	if ctx.Err() == nil {
		resultChan <- &ResultTransfer{ID: result}
	}
}
