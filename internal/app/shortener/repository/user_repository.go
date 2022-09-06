package repository

import (
	"context"
	"strconv"

	"fmt"

	"net/url"
	"sync"
)

type SyncMapUserRepo struct {
	sMap   sync.Map
	m      sync.Mutex
	lastId int
}

func InitUserRepository(backUpPath string) (Repository, error) {
	smr := new(SyncMapUserRepo)
	return smr, nil
}

func (smr *SyncMapUserRepo) Read(ctx context.Context, id string) (any, error) {
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

func (smr *SyncMapUserRepo) Create(ctx context.Context, v any) (string, error) {
	resultChan := make(chan *resultTransfer, 1)
	u, ok := v.([]*url.URL)
	if u != nil && !ok {
		return "", repositoryTypeError(v)
	}
	go smr.writeToDB(resultChan, u)
	select {
	case res := <-resultChan:
		return res.id, res.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (smr *SyncMapUserRepo) Update(ctx context.Context, id string, v any) error {
	resultChan := make(chan *resultTransfer, 1)
	u, ok := v.([]*url.URL)
	if !ok {
		return repositoryTypeError(v)
	}
	go smr.updateInDB(resultChan, id, u)
	select {
	case res := <-resultChan:
		return res.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (smr *SyncMapUserRepo) getFromDB(urlChan chan<- *valueTransfer, id string) {
	var err error = nil
	sliceOfURL, ok := smr.sMap.Load(id)
	if !ok {
		urlChan <- &valueTransfer{
			value: nil,
			err:   ErrNoSuchValue,
		}
		return
	}
	typedSliceOfURL, ok := sliceOfURL.([]*url.URL)
	if !ok {
		urlChan <- &valueTransfer{
			value: nil,
			err:   ErrUnexpectedTypeInMap,
		}
		return
	}
	urlChan <- &valueTransfer{
		value: typedSliceOfURL,
		err:   err,
	}

}

func (smr *SyncMapUserRepo) writeToDB(resultChan chan<- *resultTransfer, v []*url.URL) {
	smr.m.Lock()
	id := strconv.Itoa(smr.lastId)
	smr.lastId++
	smr.m.Unlock()
	if v != nil {
		smr.sMap.Store(id, v)
	}
	smr.sMap.Store(id, []*url.URL{})
	resultChan <- &resultTransfer{
		id: id,
	}
}
func (smr *SyncMapUserRepo) updateInDB(resultChan chan<- *resultTransfer, id string, u any) {
	smr.sMap.Store(id, u)
	resultChan <- &resultTransfer{
		id: id,
	}
}
