package controlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"io"
	"log"
	"net/url"
	"os"
	"sync"
	"time"
)

var rep repository.URLShortenerRepository
var backUpEncoder *json.Encoder
var m sync.Mutex

type backUpValue struct {
	Key   string
	Value *url.URL
}

func InitRepository(repo repository.URLShortenerRepository, backUpPath string) {
	if repo != nil {
		rep = repo
	} else {
		rep = &repository.MapBd{}
	}
	if len(backUpPath) != 0 {
		file, err := os.OpenFile(backUpPath, os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModePerm)
		if err != nil {
			log.Println(err)
			return
		}
		decoder := json.NewDecoder(file)
		backUpValue := backUpValue{}
		var decoderError error
		for decoderError = decoder.Decode(&backUpValue); decoderError == nil; decoderError = decoder.Decode(&backUpValue) {
			rep.SetKeyValue(backUpValue.Key, backUpValue.Value)
		}
		if errors.Is(err, io.EOF) {
			log.Println(err)
		}

		backUpEncoder = json.NewEncoder(file)
	}
}

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
	err := writeToBackUp(resultTransfer, unShortenURL)
	if err != nil {
		log.Println("error on write to backup" + err.Error())
	}
	return resultTransfer.ID, resultTransfer.Err
}

func writeToBackUp(transfer *repository.ResultTransfer, u *url.URL) error {
	if transfer.Err == nil && !transfer.IsDuplicate && backUpEncoder != nil {
		m.Lock()
		defer m.Unlock()
		err := backUpEncoder.Encode(backUpValue{
			Key:   transfer.ID,
			Value: u,
		})
		return err
	}
	return nil
}
