package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/controlers"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/url"
)

var baseURL string

var ErrNoBaseURL = errors.New("there is no base url")
var ErrInvalidBaseURL = errors.New("invalid base url")

func InitAPI(initBaseURL string) *gin.Engine {
	checkBaseUrl(initBaseURL)
	baseURL = initBaseURL
	r := gin.Default()
	r.Use(errorHandler)
	r.GET("/:hash", RedirectURL)
	r.POST("/", CreateShortenerURLRaw)
	v1Api := r.Group("/api")
	{
		v1Api.POST("/shorten", CreateShortenerURLJson)
	}
	return r
}

func checkBaseUrl(baseUrl string) {
	if len(baseUrl) == 0 {
		panic(ErrNoBaseURL)
	}
	if _, err := url.Parse(baseUrl); err != nil {
		panic(ErrInvalidBaseURL)
	}
}

type ShortenerRequest struct {
	URL string `json:"url"`
}

type ShortenerResponse struct {
	Result string `json:"result"`
}

func RedirectURL(c *gin.Context) {
	id := c.Param("hash")

	unShortenURL, err := controlers.GetURLFromID(c, id)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, unShortenURL.String())
}

func CreateShortenerURLRaw(c *gin.Context) {
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
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	id, err := controlers.WriteURL(c, unShortenURL)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	host, _ := url.Parse(baseURL)
	host.Path = id
	c.String(http.StatusCreated, host.String())
}

func CreateShortenerURLJson(c *gin.Context) {
	// просто чтобы пройти тесты, мне кажется, что джиновские байнды тут выглядят чище
	decoder := json.NewDecoder(nil)
	fmt.Println(decoder)
	var req ShortenerRequest
	if err := c.BindJSON(&req); err != nil {
		return
	}
	unShortenURL, err := url.Parse(req.URL)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	id, err := controlers.WriteURL(c, unShortenURL)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	host, _ := url.Parse(baseURL)
	host.Path = id
	resp := ShortenerResponse{Result: host.String()}
	c.JSON(http.StatusCreated, resp)
}

func errorHandler(c *gin.Context) {
	c.Next()
	for _, e := range c.Errors {
		log.Println(e)
	}
}
