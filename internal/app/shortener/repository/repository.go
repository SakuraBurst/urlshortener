package repository

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"sync"
)

type MapBd struct {
	sync.Map
	sync.Mutex
	backUpEncoder *json.Encoder
}

var ErrNoSuchURL = errors.New("there is no such url")
var ErrUnexpectedTypeInMap = errors.New("unexpected type in map")

type URLShortenerRepository interface {
	ReadFromBd(context.Context, string) *URLTransfer
	WriteToBd(context.Context, *url.URL) *ResultTransfer
	InitRepository(string)
}

type URLTransfer struct {
	UnShorterURL *url.URL
	Err          error
}

type backUpValue struct {
	Key   string
	Value *url.URL
}

type ResultTransfer struct {
	ID  string
	Err error
}

func (m *MapBd) InitRepository(backUpPath string) {
	if len(backUpPath) != 0 {
		file, err := os.OpenFile(backUpPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			log.Println(err)
			return
		}
		decoder := json.NewDecoder(file)
		backUpValue := backUpValue{}
		var decoderError error
		for decoderError = decoder.Decode(&backUpValue); decoderError == nil; decoderError = decoder.Decode(&backUpValue) {
			m.Store(backUpValue.Key, backUpValue.Value)
		}
		if errors.Is(err, io.EOF) {
			log.Println(err)
		}

		m.backUpEncoder = json.NewEncoder(file)
	}
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
	key := fmt.Sprintf("%x", h.Sum(nil))[:5]
	_, ok := m.LoadOrStore(key, unShortenURL)
	if !ok && m.backUpEncoder != nil {
		m.Lock()
		defer m.Unlock()
		err := m.backUpEncoder.Encode(backUpValue{
			Key:   key,
			Value: unShortenURL,
		})
		if err != nil {
			if ctx.Err() == nil {
				resultChan <- &ResultTransfer{Err: err}
			}
			return
		}
	}
	if ctx.Err() == nil {
		resultChan <- &ResultTransfer{ID: key}
	}
}
