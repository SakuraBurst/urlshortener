package controllers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"net/url"
	"time"
)

type Controller struct {
	rep repository.Repository
}

func InitController(rep repository.Repository) *Controller {
	return &Controller{rep: rep}
}

func (c *Controller) GetURLFromID(ctx context.Context, id string) (*url.URL, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	v, err := c.rep.Read(ctx, id)
	if err != nil {
		return nil, err
	}
	u, ok := v.(*url.URL)
	if !ok {
		return nil, errors.New("something went wrong")
	}
	return u, nil
}

func (c *Controller) WriteURL(ctx context.Context, unShortenURL *url.URL) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	return c.rep.Create(ctx, unShortenURL)
}

func (c *Controller) isValidToken(token string, secretKey []byte) bool {
	return false
}

func (c *Controller) createToken(id string, secretKey []byte) string {
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(id))
	return hex.EncodeToString(h.Sum(nil))
}
