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

type SyncMapURLRepo struct {
	sMap          sync.Map
	m             sync.Mutex
	backUpEncoder *json.Encoder
}

var ErrNoSuchValue = errors.New("there is no such value in repo")
var ErrUnexpectedTypeInMap = errors.New("unexpected type in map")

type Repository interface {
	Create(context.Context, any) (string, error)
	Read(context.Context, string) (any, error)
	Update(context.Context, string, any) error
}

type valueTransfer struct {
	value any
	err   error
}

type backUpValue struct {
	Key   string
	Value *url.URL
}

type resultIDTransfer struct {
	id  string
	err error
}

func TypeError(v any) error {
	return fmt.Errorf("repository dont support this type of value - %T", v)
}

func InitURLRepository(backUpPath string) (Repository, error) {
	smr := new(SyncMapURLRepo)
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

func (smr *SyncMapURLRepo) Read(ctx context.Context, id string) (any, error) {
	valueChan := make(chan *valueTransfer, 1)
	go smr.getFromDB(valueChan, id)
	select {
	case urlTransfer := <-valueChan:
		return urlTransfer.value, urlTransfer.err
	case <-ctx.Done():
		fmt.Println(ctx, ctx.Err())
		return nil, ctx.Err()
	}
}

func (smr *SyncMapURLRepo) Create(ctx context.Context, v any) (string, error) {
	resultChan := make(chan *resultIDTransfer, 1)
	u, ok := v.(*url.URL)
	if !ok {
		return "", TypeError(v)
	}
	go smr.writeToDB(resultChan, u)
	select {
	case res := <-resultChan:
		return res.id, res.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (smr *SyncMapURLRepo) Update(ctx context.Context, id string, v any) error {
	resultChan := make(chan *resultIDTransfer, 1)
	u, ok := v.(*url.URL)
	if !ok {
		return TypeError(v)
	}
	go smr.updateInDB(resultChan, id, u)
	select {
	case res := <-resultChan:
		return res.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (smr *SyncMapURLRepo) getFromDB(valueChan chan<- *valueTransfer, id string) {
	var err error = nil
	untypedURL, ok := smr.sMap.Load(id)
	if !ok {
		valueChan <- &valueTransfer{
			value: nil,
			err:   ErrNoSuchValue,
		}
		return
	}
	unShorterURL, ok := untypedURL.(*url.URL)
	if !ok {
		valueChan <- &valueTransfer{
			value: nil,
			err:   ErrUnexpectedTypeInMap,
		}
		return
	}
	valueChan <- &valueTransfer{
		value: unShorterURL,
		err:   err,
	}

}

func (smr *SyncMapURLRepo) writeToDB(resultChan chan<- *resultIDTransfer, unShortenURL *url.URL) {
	h := sha1.New()
	_, err := io.WriteString(h, unShortenURL.String())
	if err != nil {
		resultChan <- &resultIDTransfer{err: err}
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
			resultChan <- &resultIDTransfer{err: err}
			return
		}
	}
	resultChan <- &resultIDTransfer{id: key}
}
func (smr *SyncMapURLRepo) updateInDB(resultChan chan<- *resultIDTransfer, id string, u any) {
	smr.sMap.Store(id, u)
	resultChan <- &resultIDTransfer{
		id: id,
	}
}
