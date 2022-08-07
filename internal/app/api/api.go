package api

import (
	"errors"
	"github.com/SakuraBurst/urlshortener/internal/controlers"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/url"
)

var host = url.URL{
	Scheme: "http",
	Host:   "localhost:8080",
}

func InitAPI() *gin.Engine {
	r := gin.Default()
	r.Use(errorHandler)
	r.GET("/:hash", RedirectURL)
	r.POST("/", CreateShortenerURL)
	return r
}

func RedirectURL(c *gin.Context) {
	id := c.Param("hash")

	unShortenURL, err := controlers.GetURLFromID(id)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, unShortenURL.String())
}

func CreateShortenerURL(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if len(body) == 0 {
		c.AbortWithError(http.StatusBadRequest, errors.New("request body is empty"))
		return
	}
	unShortenURL, err := url.Parse(string(body))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	id, err := controlers.WriteURL(unShortenURL)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	host.Path = "/" + id
	defer func() {
		host.Path = ""
	}()
	c.String(http.StatusCreated, host.String())
	if err != nil {
		log.Println(err)
	}
}

func errorHandler(c *gin.Context) {
	c.Next()
	for _, e := range c.Errors {
		log.Println(e)
	}
}
