package api

import (
	"bytes"
	"github.com/SakuraBurst/urlshortener/internal/controlers"
	"io"
	"log"
	"net/http"
	"net/url"
)

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
	id := request.URL.Query().Get("id")
	unShortenURL, err := controlers.GetUrlFromId(id)
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
	_, err = io.Copy(buf, request.Body)
	if err != nil {
		errorHandler(writer, http.StatusInternalServerError, err)
		return
	}
	unShortenURL, err := url.Parse(buf.String())
	if err != nil {
		errorHandler(writer, http.StatusInternalServerError, err)
		return
	}
	id, err := controlers.WriteUrl(unShortenURL)
	if err != nil {
		errorHandler(writer, http.StatusInternalServerError, err)
		return
	}
	writer.WriteHeader(http.StatusCreated)
	_, err = writer.Write([]byte(id))
	if err != nil {
		log.Println(err)
	}
}

func errorHandler(writer http.ResponseWriter, statusCode int, err error) {
	log.Println(err)
	writer.WriteHeader(statusCode)
}
