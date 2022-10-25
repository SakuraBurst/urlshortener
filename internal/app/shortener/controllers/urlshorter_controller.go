package controllers

import (
	"context"
	"emperror.dev/errors"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/token"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/types"
	"github.com/jackc/pgx/v4"
	"golang.org/x/exp/slices"
	"log"
	"net/url"
	"time"
)

type Controller struct {
	urlRep       repository.Repository
	userRep      repository.Repository
	baseURL      string
	db           *pgx.Conn
	tokenBuilder *token.TokenBuilder
}

var ErrNoBaseURL = errors.New("there is no base url")
var ErrInvalidBaseURL = errors.New("invalid base url")

func InitController(initBaseURL string, db *pgx.Conn, tb *token.TokenBuilder, urlRep, userRep repository.Repository) *Controller {
	checkBaseURL(initBaseURL)
	return &Controller{baseURL: initBaseURL, urlRep: urlRep, userRep: userRep, db: db, tokenBuilder: tb}
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

func (c *Controller) WriteURL(ctx context.Context, unShortenURL *url.URL, userToken string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	id, err := c.urlRep.Create(ctx, unShortenURL)
	hasConflicts := false
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			hasConflicts = true
		} else {
			return "", false, err
		}
	}
	host, err := url.Parse(c.baseURL)
	if err != nil {
		log.Fatal("unexpected base url parse error")
	}
	host.Path = id
	return host.String(), hasConflicts, c.UpdateUser(ctx, userToken, id)
}

func (c *Controller) WriteArrayOfURL(ctx context.Context, unShortenURLs []*url.URL, userToken string) ([]string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	ids, err := c.urlRep.CreateArray(ctx, unShortenURLs)
	hasConflicts := false
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			hasConflicts = true
		} else {
			return nil, false, err
		}
	}
	host, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, false, err
	}
	urls := make([]string, 0, len(ids))
	for _, id := range ids {
		host.Path = id
		urls = append(urls, host.String())
	}
	return urls, hasConflicts, c.UpdateUser(ctx, userToken, ids...)
}

func (c *Controller) CreateUser(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	var initURLSlice []string
	id, err := c.userRep.Create(ctx, initURLSlice)
	if err != nil {
		return "", err
	}
	return c.tokenBuilder.CreateToken(id)
}

func (c *Controller) UpdateUser(ctx context.Context, userToken string, urlIDs ...string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	userID, err := c.tokenBuilder.GetIDFromToken(userToken)
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
	for _, id := range urlIDs {
		if slices.Contains(u, id) {
			continue
		}
		u = append(u, id)
	}
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
	userID, err := c.tokenBuilder.GetIDFromToken(userToken)
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
			ID:          id,
			ShortURL:    host.String(),
			OriginalURL: r.String(),
		})
	}
	return res, nil
}

func (c *Controller) DeleteArrayOfIds(ids []string, userToken string) {
	userID, err := c.tokenBuilder.GetIDFromToken(userToken)
	if err != nil {
		log.Print(err)
		return
	}
	v, err := c.userRep.Read(context.Background(), userID)
	if err != nil {
		log.Print(err)
		return
	}
	u, ok := v.([]string)
	if u != nil && !ok {
		return
	}
	res := make([]any, 0, len(ids))
	for _, id := range ids {
		if slices.Contains(u, id) {
			res = append(res, id)
		}
	}
	err = c.urlRep.Delete(context.Background(), res...)
	if err != nil {
		log.Print(err)
	}
}

func checkBaseURL(baseURL string) {
	if len(baseURL) == 0 {
		panic(ErrNoBaseURL)
	}
	if _, err := url.Parse(baseURL); err != nil {
		panic(ErrInvalidBaseURL)
	}
}
