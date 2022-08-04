package api

import (
	"bytes"
	"errors"
	"github.com/SakuraBurst/urlshortener/internal/controlers"
	"io"
	"log"
	"net/http"
	"net/url"
)

var host = url.URL{
	Scheme: "http",
	Host:   "localhost:8080",
}

func InitAPI() {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			RedirectURL(writer, request)

		case http.MethodPost:
			CreateShortenerURL(writer, request)

		default:
			writer.WriteHeader(http.StatusBadRequest)
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func RedirectURL(writer http.ResponseWriter, request *http.Request) {
	id := request.URL.Path[1:]
	if len(id) == 0 {
		errorHandler(writer, http.StatusBadRequest, errors.New("there is no id in query"))
		return
	}

	unShortenURL, err := controlers.GetURLFromID(id)
	if err != nil {
		errorHandler(writer, http.StatusNotFound, err)
		return
	}
	writer.Header().Set("location", unShortenURL.String())
	writer.WriteHeader(http.StatusTemporaryRedirect)
}

func CreateShortenerURL(writer http.ResponseWriter, request *http.Request) {
	buf := bytes.NewBuffer(nil)
	_, err := io.Copy(buf, request.Body)
	if err != nil {
		errorHandler(writer, http.StatusInternalServerError, err)
		return
	}
	defer request.Body.Close()
	if buf.Len() == 0 {
		errorHandler(writer, http.StatusBadRequest, err)
		return
	}
	unShortenURL, err := url.Parse(buf.String())
	if err != nil {
		errorHandler(writer, http.StatusInternalServerError, err)
		return
	}
	id, err := controlers.WriteURL(unShortenURL)
	if err != nil {
		errorHandler(writer, http.StatusInternalServerError, err)
		return
	}
	host.Path = "/" + id
	defer func() {
		host.Path = ""
	}()
	writer.Header().Set("content-type", "text/plain")
	writer.WriteHeader(http.StatusCreated)
	_, err = writer.Write([]byte(host.String()))
	if err != nil {
		log.Println(err)
	}
}

func errorHandler(writer http.ResponseWriter, statusCode int, err error) {
	log.Println(err)
	writer.WriteHeader(statusCode)
}
