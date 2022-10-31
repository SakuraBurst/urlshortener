package repository

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"io"
	"log"
	"net/url"
	"os"
	"sync"
)

type DBURLRepo struct {
	db         *pgx.Conn
	insertStmt *pgconn.StatementDescription
}

func initURLRepository(c context.Context, backUpPath string, db *pgx.Conn) (Repository, error) {
	if db != nil {
		r := db.QueryRow(c, "SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename  = 'url')")
		var isExist bool
		err := r.Scan(&isExist)
		if err != nil {
			return nil, err
		}
		if !isExist {
			_, err := db.Exec(c, "create table url (shortenHash text primary key, unShortenURL text, isDeleted boolean)")
			if err != nil {
				return nil, err
			}
		}
		stmt, err := db.Prepare(c, "insert url", "INSERT INTO url (shortenhash, unshortenurl, isDeleted) VALUES ($1, $2, false) on conflict do nothing RETURNING shortenHash")
		if err != nil {
			return nil, err
		}
		return &DBURLRepo{db: db, insertStmt: stmt}, nil
	}
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

func (d *DBURLRepo) Create(ctx context.Context, v any) (string, error) {
	u, ok := v.(*url.URL)
	if !ok {
		return "", TypeError(v)
	}
	key, err := createURLHash(u)
	if err != nil {
		return "", err
	}
	r := d.db.QueryRow(ctx, d.insertStmt.SQL, key, u.String())
	hash := ""
	err = r.Scan(&hash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return key, ErrDuplicate
	}
	return key, err
}

func (d *DBURLRepo) CreateArray(ctx context.Context, v any) ([]string, error) {
	urls, ok := v.([]*url.URL)
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
	stmt, err := tx.Prepare(ctx, "inset url tx", d.insertStmt.SQL)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(urls))
	isDuplicate := false
	for _, u := range urls {
		key, err := createURLHash(u)
		if err != nil {
			return nil, err
		}
		r := tx.QueryRow(ctx, stmt.SQL, key, u.String())
		hash := ""
		err = r.Scan(&hash)
		if errors.Is(err, pgx.ErrNoRows) {
			isDuplicate = true
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		result = append(result, key)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	if isDuplicate {
		return result, ErrDuplicate
	}
	return result, err
}

func (d *DBURLRepo) Read(ctx context.Context, id string) (any, error) {
	r := d.db.QueryRow(ctx, "SELECT unshortenurl, isDeleted from url where shortenhash = $1", id)
	s := ""
	isDeleted := false
	err := r.Scan(&s, &isDeleted)
	if err != nil {
		return nil, err
	}
	if isDeleted {
		return nil, ErrDeleted
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

func (d *DBURLRepo) Delete(ctx context.Context, s ...string) error {
	return d.batchUpdate(ctx, s, "UPDATE url set isDeleted = true WHERE shortenhash = $1")
}

type sql struct {
	sql       string
	arguments []interface{}
}

func (d *DBURLRepo) batchUpdate(ctx context.Context, ids []string, sqlString string) error {
	mc := make(chan string)
	numberOfWorkers := len(ids) % 20
	in := fanIn(mc, numberOfWorkers)
	out := make([]chan *sql, 0, numberOfWorkers)
	for i := 0; i < numberOfWorkers; i++ {
		c := make(chan *sql)
		out = append(out, c)
		builder(sqlString, in[i], c)
	}
	resC := fanOut(out...)
	go func() {
		for _, id := range ids {
			mc <- id
		}
	}()
	return d.batchWorker(ctx, resC)
}

func fanIn(inputChan <-chan string, n int) []chan string {
	a := make([]chan string, 0, n)
	for i := 0; i < n; i++ {
		a = append(a, make(chan string))
	}
	go func() {
		defer func() {
			for _, c := range a {
				close(c)
			}
		}()
		for i := 0; ; i++ {
			if i == n {
				i = 0
			}
			v, ok := <-inputChan
			if !ok {
				return
			}
			a[i] <- v
		}
	}()
	return a
}

func builder(sqlString string, in <-chan string, out chan<- *sql) {
	go func() {
		for id := range in {
			out <- &sql{
				sql:       sqlString,
				arguments: []interface{}{id},
			}
		}
		close(out)
	}()
}

func fanOut(in ...chan *sql) chan *sql {
	output := make(chan *sql)
	go func() {
		wg := sync.WaitGroup{}
		for _, sqls := range in {
			wg.Add(1)
			go func(sqls chan *sql) {
				defer wg.Done()
				for s := range sqls {
					output <- s
				}
			}(sqls)
		}
		wg.Wait()
		close(output)
	}()

	return output
}

func (d *DBURLRepo) batchWorker(ctx context.Context, in chan *sql) error {
	batch := &pgx.Batch{}

	for {
		if batch.Len() > 2 {
			br := d.db.SendBatch(ctx, batch)
			if err := br.Close(); err != nil {
				return err
			}
			batch = &pgx.Batch{}
		}
		stmt, ok := <-in
		if !ok {
			break
		}
		batch.Queue(stmt.sql, stmt.arguments...)
	}
	if batch.Len() > 0 {
		br := d.db.SendBatch(ctx, batch)
		if err := br.Close(); err != nil {
			return err
		}
	}
	return nil
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
		return nil, ctx.Err()
	}
}

func (smr *SyncMapURLRepo) Create(ctx context.Context, v any) (string, error) {
	resultChan := make(chan *resultIDTransfer, 1)
	u, ok := v.(*url.URL)
	if !ok {
		return "", TypeError(v)
	}
	go smr.writeToDB(resultChan, u, 0)
	select {
	case res := <-resultChan:
		return res.id, res.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (smr *SyncMapURLRepo) CreateArray(ctx context.Context, v any) ([]string, error) {
	urls, ok := v.([]*url.URL)
	if !ok {
		return nil, TypeError(v)
	}
	resultChan := make(chan *resultIDTransfer, len(urls))
	for i, u := range urls {
		go smr.writeToDB(resultChan, u, i)
	}
	result := make([]string, len(urls))
	for range result {
		select {
		case res := <-resultChan:
			if res.err != nil {
				return nil, res.err
			}
			result[res.index] = res.id
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return result, nil
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

func (smr *SyncMapURLRepo) Delete(ctx context.Context, id ...string) error {
	panic("unsupported behavior")
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

func (smr *SyncMapURLRepo) writeToDB(resultChan chan<- *resultIDTransfer, unShortenURL *url.URL, index int) {
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
	resultChan <- &resultIDTransfer{id: key, index: index}
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
