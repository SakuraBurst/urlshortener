package controlers

import (
	"context"
	"github.com/SakuraBurst/urlshortener/internal/repository"
	"net/url"
	"time"
)

var rep repository.URLShortenerRepository = repository.MapBd{}

func SetRepository(repo repository.URLShortenerRepository) {
	rep = repo
}

func GetURLFromID(ctx context.Context, id string) (*url.URL, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()
	urlTransfer := rep.Read(ctx, id)
	return urlTransfer.UnShorterURL, urlTransfer.Err
}

func WriteURL(ctx context.Context, unShortenURL *url.URL) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	resultTransfer := rep.Write(ctx, unShortenURL)
	return resultTransfer.ID, resultTransfer.Err
}
