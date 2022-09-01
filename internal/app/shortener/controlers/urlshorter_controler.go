package controlers

import (
	"context"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"net/url"
	"time"
)

var rep repository.URLShortenerRepository

func SetRepository(repo repository.URLShortenerRepository) {
	rep = repo
}

func GetURLFromID(ctx context.Context, id string) (*url.URL, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	urlTransfer := rep.ReadFromBd(ctx, id)
	return urlTransfer.UnShorterURL, urlTransfer.Err
}

func WriteURL(ctx context.Context, unShortenURL *url.URL) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	resultTransfer := rep.WriteToBd(ctx, unShortenURL)
	return resultTransfer.ID, resultTransfer.Err
}
