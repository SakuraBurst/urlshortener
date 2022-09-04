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

type SyncMapRepo struct {
	sMap          sync.Map
	m             sync.Mutex
	backUpEncoder *json.Encoder
}

var ErrNoSuchURL = errors.New("there is no such url")
var ErrUnexpectedTypeInMap = errors.New("unexpected type in map")

type URLShortenerRepository interface {
	Read(context.Context, string) (*url.URL, error)
	Write(context.Context, *url.URL) (string, error)
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

func InitRepository(backUpPath string) (*SyncMapRepo, error) {
	smr := new(SyncMapRepo)
	if len(backUpPath) != 0 {
		file, err := os.OpenFile(backUpPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		decoder := json.NewDecoder(file)
		backUpVal := backUpValue{}
		var decoderError error
		for decoderError = decoder.Decode(&backUpVal); decoderError == nil; decoderError = decoder.Decode(&backUpVal) {
			smr.sMap.Store(backUpVal.Key, backUpVal.Value)
			backUpVal = backUpValue{}
		}
		if errors.Is(err, io.EOF) {
			log.Println(err)
		}

		smr.backUpEncoder = json.NewEncoder(file)
	}
	return smr, nil
}

func (smr *SyncMapRepo) Read(ctx context.Context, id string) (*url.URL, error) {
	urlChan := make(chan *URLTransfer, 1)
	go smr.getFromDB(urlChan, id)
	select {
	case urlTransfer := <-urlChan:
		return urlTransfer.UnShorterURL, urlTransfer.Err
	case <-ctx.Done():
		fmt.Println(ctx, ctx.Err())
		return nil, ctx.Err()
	}
}

func (smr *SyncMapRepo) Write(ctx context.Context, u *url.URL) (string, error) {
	fmt.Println("?")
	resultChan := make(chan *ResultTransfer, 1)
	go smr.writeToDB(resultChan, u)
	select {
	case res := <-resultChan:
		return res.ID, res.Err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (smr *SyncMapRepo) getFromDB(urlChan chan<- *URLTransfer, id string) {
	var err error = nil
	untypedURL, ok := smr.sMap.Load(id)
	if !ok {
		urlChan <- &URLTransfer{
			UnShorterURL: nil,
			Err:          ErrNoSuchURL,
		}
		return
	}
	var unShorterURL *url.URL
	switch v := untypedURL.(type) {
	case *url.URL:
		unShorterURL = v
	default:
		urlChan <- &URLTransfer{
			UnShorterURL: nil,
			Err:          ErrUnexpectedTypeInMap,
		}
		return
	}
	urlChan <- &URLTransfer{
		UnShorterURL: unShorterURL,
		Err:          err,
	}

}

func (smr *SyncMapRepo) writeToDB(resultChan chan<- *ResultTransfer, unShortenURL *url.URL) {
	h := sha1.New()
	_, err := io.WriteString(h, unShortenURL.String())
	if err != nil {
		resultChan <- &ResultTransfer{Err: err}
		return
	}
	key := fmt.Sprintf("%x", h.Sum(nil))[:5]
	_, ok := smr.sMap.LoadOrStore(key, unShortenURL)
	if !ok && smr.backUpEncoder != nil {
		smr.m.Lock()
		defer smr.m.Unlock()
		err := smr.backUpEncoder.Encode(backUpValue{
			Key:   key,
			Value: unShortenURL,
		})
		if err != nil {
			resultChan <- &ResultTransfer{Err: err}

			return
		}
	}
	resultChan <- &ResultTransfer{ID: key}

}
