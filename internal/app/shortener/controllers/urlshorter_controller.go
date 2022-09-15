package controllers

import (
	"context"
	"database/sql"
	"errors"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/token"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/types"
	"net/url"
	"time"
)

type Controller struct {
	urlRep  repository.Repository
	userRep repository.Repository
	baseURL string
	db      *sql.DB
}

var ErrNoBaseURL = errors.New("there is no base url")
var ErrInvalidBaseURL = errors.New("invalid base url")

func InitController(initBaseURL, dbURL string, urlRep, userRep repository.Repository) *Controller {
	checkBaseURL(initBaseURL)
	db, _ := sql.Open("pgx", dbURL)
	return &Controller{baseURL: initBaseURL, urlRep: urlRep, userRep: userRep, db: db}
}

func (c *Controller) GetURLFromID(ctx context.Context, id string) (*url.URL, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	v, err := c.urlRep.Read(ctx, id)
	if err != nil {
		return nil, err
	}
	u, ok := v.(*url.URL)
	if !ok {
		return nil, repository.TypeError(u)
	}
	return u, nil
}

func (c *Controller) WriteURL(ctx context.Context, unShortenURL *url.URL, userToken string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	id, err := c.urlRep.Create(ctx, unShortenURL)
	if err != nil {
		return "", err
	}
	host, _ := url.Parse(c.baseURL)
	host.Path = id
	err = c.UpdateUser(ctx, userToken, &types.URLShorter{
		ShortURL:    host.String(),
		OriginalURL: unShortenURL.String(),
	})
	return host.String(), err
}

func (c *Controller) CreateUser(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	var initURLSlice []*types.URLShorter
	id, err := c.userRep.Create(ctx, initURLSlice)
	if err != nil {
		return "", err
	}
	return token.CreateToken(id)
}

func (c *Controller) UpdateUser(ctx context.Context, userToken string, updateValue *types.URLShorter) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	userID, err := token.GetIDFromToken(userToken)
	if err != nil {
		return err
	}
	v, err := c.GetUser(ctx, userToken)
	if err != nil {
		return err
	}
	v = append(v, updateValue)
	return c.userRep.Update(ctx, userID, v)
}

func (c *Controller) PingDataBase() error {
	if c.db == nil {
		return errors.New("there is no db conn")
	}
	return c.db.Ping()
}

func (c *Controller) GetUser(ctx context.Context, userToken string) ([]*types.URLShorter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	userID, err := token.GetIDFromToken(userToken)
	if err != nil {
		return nil, err
	}
	v, err := c.userRep.Read(ctx, userID)
	if err != nil {
		return nil, err
	}
	u, ok := v.([]*types.URLShorter)
	if u != nil && !ok {
		return nil, repository.TypeError(u)
	}
	return u, nil
}

func checkBaseURL(baseURL string) {
	if len(baseURL) == 0 {
		panic(ErrNoBaseURL)
	}
	if _, err := url.Parse(baseURL); err != nil {
		panic(ErrInvalidBaseURL)
	}
}
