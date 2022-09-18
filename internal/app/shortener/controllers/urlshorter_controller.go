package controllers

import (
	"context"
	"emperror.dev/errors"
	"fmt"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/token"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/types"
	"github.com/jackc/pgx/v4"
	"golang.org/x/exp/slices"
	"net/url"
	"time"
)

type Controller struct {
	urlRep  repository.Repository
	userRep repository.Repository
	baseURL string
	db      *pgx.Conn
}

var ErrNoBaseURL = errors.New("there is no base url")
var ErrInvalidBaseURL = errors.New("invalid base url")

func InitController(initBaseURL string, db *pgx.Conn, urlRep, userRep repository.Repository) *Controller {
	checkBaseURL(initBaseURL)
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
	err = c.UpdateUser(ctx, userToken, id)
	return host.String(), err
}

func (c *Controller) CreateUser(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	var initURLSlice []string
	fmt.Println("dd")
	id, err := c.userRep.Create(ctx, initURLSlice)
	if err != nil {
		return "", err
	}
	return token.CreateToken(id)
}

func (c *Controller) UpdateUser(ctx context.Context, userToken, urlID string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	userID, err := token.GetIDFromToken(userToken)
	if err != nil {
		return err
	}
	v, err := c.userRep.Read(ctx, userID)
	if err != nil {
		return err
	}
	u, ok := v.([]string)
	if u != nil && !ok {
		return repository.TypeError(u)
	}
	if slices.Contains(u, urlID) {
		return nil
	}
	u = append(u, urlID)
	return c.userRep.Update(ctx, userID, u)
}

func (c *Controller) PingDataBase(ctx context.Context) error {
	if c.db == nil {
		return errors.New("there is no db conn")
	}
	return c.db.Ping(ctx)
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
	u, ok := v.([]string)
	if u != nil && !ok {
		return nil, repository.TypeError(u)
	}
	res := make([]*types.URLShorter, 0, len(u))
	for _, id := range u {
		host, _ := url.Parse(c.baseURL)
		host.Path = id
		r, err := c.GetURLFromID(ctx, id)
		if err != nil {
			return nil, err
		}
		res = append(res, &types.URLShorter{
			ShortURL:    host.String(),
			OriginalURL: r.String(),
		})

	}
	return res, nil
}

func checkBaseURL(baseURL string) {
	if len(baseURL) == 0 {
		panic(ErrNoBaseURL)
	}
	if _, err := url.Parse(baseURL); err != nil {
		panic(ErrInvalidBaseURL)
	}
}
