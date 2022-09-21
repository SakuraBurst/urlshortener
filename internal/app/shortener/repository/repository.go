package repository

import (
	"context"
	"emperror.dev/errors"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v4"
	"io"
	"log"
	"net/url"
	"os"
)

var ErrNoSuchValue = errors.New("there is no such value in repo")
var ErrUnexpectedTypeInMap = errors.New("unexpected type in map")

type Repository interface {
	Create(context.Context, any) (string, error)
	CreateArray(context.Context, any) ([]string, error)
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
	id    string
	index int
	err   error
}

func TypeError(v any) error {
	return fmt.Errorf("repository dont support this type of value - %T", v)
}

func initUserRepository(c context.Context, db *pgx.Conn) (Repository, error) {
	if db != nil {
		r := db.QueryRow(c, "SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename  = 'users')")
		var isExist bool
		err := r.Scan(&isExist)
		if err != nil {
			return nil, err
		}
		if !isExist {
			_, err = db.Exec(c, "create table users (id serial primary key , urls text [])")
			if err != nil {
				return nil, err
			}
		}
		stmt, err := db.Prepare(c, "insert user", `INSERT INTO users (urls) values ($1) RETURNING id`)
		if err != nil {
			return nil, err
		}
		return &DBUserRepo{db: db, insertStmt: stmt}, nil
	}
	smr := &SyncMapUserRepo{lastID: 1}
	return smr, nil
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
			_, err := db.Exec(c, "create table url (shortenHash text primary key, unShortenURL text)")
			if err != nil {
				return nil, err
			}
		}
		stmt, err := db.Prepare(c, "insert url", "INSERT INTO url (shortenhash, unshortenurl) VALUES ($1, $2) ON CONFLICT DO NOTHING")
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

func InitRepositories(c context.Context, backUpPath string, db *pgx.Conn) (urlRepo, userRepo Repository, err error) {
	urlRepo, err = initURLRepository(c, backUpPath, db)
	if err != nil {
		return
	}
	userRepo, err = initUserRepository(c, db)
	return
}
