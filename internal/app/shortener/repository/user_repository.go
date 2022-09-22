package repository

import (
	"context"
	"emperror.dev/errors"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"log"
	"strconv"

	"sync"
)

type SyncMapUserRepo struct {
	sMap   sync.Map
	m      sync.Mutex
	lastID int
}

type DBUserRepo struct {
	db         *pgx.Conn
	insertStmt *pgconn.StatementDescription
}

func (d *DBUserRepo) Create(ctx context.Context, v any) (string, error) {
	u, ok := v.([]string)
	if u != nil && !ok {
		return "", TypeError(v)
	}
	res := d.db.QueryRow(ctx, d.insertStmt.SQL, u)
	id := 0
	err := res.Scan(&id)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(id), nil
}

func (d *DBUserRepo) CreateArray(ctx context.Context, v any) ([]string, error) {
	usersURLs, ok := v.([][]string)
	if !ok {
		return nil, TypeError(v)
	}
	tx, err := d.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		err := tx.Rollback(ctx)
		if err != nil {
			log.Println(err)
		}
	}()
	stmt, err := tx.Prepare(ctx, "inset user tx", d.insertStmt.SQL)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(usersURLs))
	for _, userURLs := range usersURLs {
		row := tx.QueryRow(ctx, stmt.SQL, userURLs)
		id := 0
		err := row.Scan(&id)
		if err != nil {
			return nil, err
		}
		result = append(result, strconv.Itoa(id))
	}
	return result, tx.Commit(ctx)
}

func (d *DBUserRepo) Read(ctx context.Context, s string) (any, error) {
	row := d.db.QueryRow(ctx, "select urls from users where id = $1", s)
	res := make([]string, 0)
	err := row.Scan(&res)
	if err != nil {
		return "", err
	}
	return res, nil
}

func (d *DBUserRepo) Update(ctx context.Context, s string, v any) error {
	u, ok := v.([]string)
	if u != nil && !ok {
		return TypeError(v)
	}
	_, err := d.db.Exec(ctx, "UPDATE users set urls = $1 where id = $2", u, s)
	return err
}

func (smr *SyncMapUserRepo) Read(ctx context.Context, id string) (any, error) {
	valueChan := make(chan *valueTransfer, 1)
	go smr.getFromDB(valueChan, id)
	select {
	case urlTransfer := <-valueChan:
		return urlTransfer.value, urlTransfer.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (smr *SyncMapUserRepo) Create(ctx context.Context, v any) (string, error) {
	resultChan := make(chan *resultIDTransfer, 1)
	u, ok := v.([]string)
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

func (smr *SyncMapUserRepo) CreateArray(ctx context.Context, v any) ([]string, error) {
	usersURLs, ok := v.([][]string)
	if !ok {
		return nil, TypeError(v)
	}
	resultChan := make(chan *resultIDTransfer, len(usersURLs))
	for _, u := range usersURLs {
		go smr.writeToDB(resultChan, u)
	}
	result := make([]string, len(usersURLs))
	for i := range result {
		select {
		case res := <-resultChan:
			if res.err != nil {
				return nil, res.err
			}
			result[i] = res.id
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, errors.New("unexpected error")
}

func (smr *SyncMapUserRepo) Update(ctx context.Context, id string, v any) error {
	resultChan := make(chan *resultIDTransfer, 1)
	u, ok := v.([]string)
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
		urlChan <- &valueTransfer{
			value: nil,
			err:   ErrNoSuchValue,
		}
		return
	}
	typedSliceOfURL, ok := sliceOfURL.([]string)
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

func (smr *SyncMapUserRepo) writeToDB(resultChan chan<- *resultIDTransfer, v []string) {
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
