package repository

import (
	"context"
	"emperror.dev/errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"net/url"
)

var ErrNoSuchValue = errors.New("there is no such value in repo")
var ErrUnexpectedTypeInMap = errors.New("unexpected type in map")
var ErrDuplicate = errors.New("there is duplicate in data")

type Repository interface {
	Create(context.Context, any) (string, error)
	CreateArray(context.Context, any) ([]string, error)
	Read(context.Context, string) (any, error)
	Update(context.Context, string, any) error
	Delete(context.Context, string) error
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

func InitRepositories(c context.Context, backUpPath string, db *pgx.Conn) (urlRepo, userRepo Repository, err error) {
	urlRepo, err = initURLRepository(c, backUpPath, db)
	if err != nil {
		return
	}
	userRepo, err = initUserRepository(c, db)
	return
}
