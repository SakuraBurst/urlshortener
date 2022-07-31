package controlers

import (
	"context"
	"github.com/SakuraBurst/urlshortener/internal/repositroy"
	"net/url"
	"time"
)

func GetUrlFromId(id string) (*url.URL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	urlTransfer := repositroy.GetFromBd(ctx, id)
	return urlTransfer.UnShorterURL, urlTransfer.Err
}

func WriteUrl(unShortenURL *url.URL) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	resultTransfer := repositroy.WriteToBd(ctx, unShortenURL)
	return resultTransfer.Id, resultTransfer.Err
}
