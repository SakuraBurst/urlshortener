package controllers

import (
	"context"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"net/url"
	"time"
)

type Controller struct {
	rep repository.URLShortenerRepository
}

func InitController(rep repository.URLShortenerRepository) *Controller {
	return &Controller{rep: rep}
}

func (c *Controller) GetURLFromID(ctx context.Context, id string) (*url.URL, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	return c.rep.Read(ctx, id)
}

func (c *Controller) WriteURL(ctx context.Context, unShortenURL *url.URL) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	return c.rep.Write(ctx, unShortenURL)
}
