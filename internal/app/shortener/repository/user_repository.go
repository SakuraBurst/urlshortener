package repository

import (
	"context"
	"fmt"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/types"
	"strconv"

	"sync"
)

type SyncMapUserRepo struct {
	sMap   sync.Map
	m      sync.Mutex
	lastID int
}

func InitUserRepository() (Repository, error) {
	smr := &SyncMapUserRepo{lastID: 1}
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
	resultChan := make(chan *resultIDTransfer, 1)
	u, ok := v.([]*types.URLShorter)
	if u != nil && !ok {
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

func (smr *SyncMapUserRepo) Update(ctx context.Context, id string, v any) error {
	resultChan := make(chan *resultIDTransfer, 1)
	u, ok := v.([]*types.URLShorter)
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

func (smr *SyncMapUserRepo) getFromDB(urlChan chan<- *valueTransfer, id string) {
	var err error
	sliceOfURL, ok := smr.sMap.Load(id)
	if !ok {
		fmt.Println(id)
		urlChan <- &valueTransfer{
			value: nil,
			err:   ErrNoSuchValue,
		}
		return
	}
	typedSliceOfURL, ok := sliceOfURL.([]*types.URLShorter)
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

func (smr *SyncMapUserRepo) writeToDB(resultChan chan<- *resultIDTransfer, v []*types.URLShorter) {
	smr.m.Lock()
	id := strconv.Itoa(smr.lastID)
	smr.lastID++
	smr.m.Unlock()
	smr.sMap.Store(id, v)

	resultChan <- &resultIDTransfer{
		id: id,
	}
}
func (smr *SyncMapUserRepo) updateInDB(resultChan chan<- *resultIDTransfer, id string, u any) {
	smr.sMap.Store(id, u)
	resultChan <- &resultIDTransfer{
		id: id,
	}
}
