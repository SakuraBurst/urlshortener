package repository

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v4"
	"io"
	"net/url"
	"sync"
)

type DBURLRepo struct {
	db *pgx.Conn
}

var errDuplicate = `ERROR: duplicate key value violates unique constraint "url_pkey" (SQLSTATE 23505)`

func (d *DBURLRepo) Create(ctx context.Context, v any) (string, error) {
	u, ok := v.(*url.URL)
	if !ok {
		return "", TypeError(v)
	}
	key, err := createURLHash(u)
	if err != nil {
		return "", err
	}
	_, err = d.db.Exec(ctx, "INSERT INTO url (shortenhash, unshortenurl) VALUES ($1, $2)", key, u.String())
	if err != nil && err.Error() != errDuplicate {
		return "", err
	}
	return key, nil
}

func (d *DBURLRepo) Read(ctx context.Context, id string) (any, error) {
	r := d.db.QueryRow(ctx, "SELECT unshortenurl from url where shortenhash = $1", id)
	s := ""
	err := r.Scan(&s)
	if err != nil {
		return nil, err
	}
	return url.Parse(s)
}

func (d *DBURLRepo) Update(ctx context.Context, s string, v any) error {
	u, ok := v.(*url.URL)
	if !ok {
		return TypeError(v)
	}
	_, err := d.db.Exec(ctx, "UPDATE url set unshortenurl = $1 where shortenhash = $2", u.String(), s)
	return err
}

type SyncMapURLRepo struct {
	sMap          sync.Map
	m             sync.Mutex
	backUpEncoder *json.Encoder
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
	key, err := createURLHash(unShortenURL)
	if err != nil {
		resultChan <- &resultIDTransfer{err: err}
		return
	}
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

func createURLHash(u *url.URL) (string, error) {
	h := sha1.New()
	_, err := io.WriteString(h, u.String())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:5], nil
}
