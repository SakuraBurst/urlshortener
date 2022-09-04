package router

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/controllers"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"net/url"
)

type router struct {
	baseURL    string
	controller *controllers.Controller
}

var ErrNoBaseURL = errors.New("there is no base url")
var ErrInvalidBaseURL = errors.New("invalid base url")

type encodeResponseWriter struct {
	gin.ResponseWriter
	Writer io.Writer
}

func (w encodeResponseWriter) Write(p []byte) (n int, err error) {
	fmt.Println(string(p))
	return w.Writer.Write(p)
}

func (w encodeResponseWriter) WriteString(s string) (n int, err error) {
	return w.Writer.Write([]byte(s))
}

func InitAPI(initBaseURL string, controller *controllers.Controller) *gin.Engine {
	checkBaseURL(initBaseURL)
	router := &router{baseURL: initBaseURL, controller: controller}
	engine := gin.Default()
	engine.Use(errorHandler)
	engine.Use(encodingHandler)
	engine.GET("/:hash", router.RedirectURL)
	engine.POST("/", router.CreateShortenerURLRaw)
	v1Api := engine.Group("/api")
	{
		v1Api.POST("/shorten", router.CreateShortenerURLJson)
	}
	return engine
}

func checkBaseURL(baseURL string) {
	if len(baseURL) == 0 {
		panic(ErrNoBaseURL)
	}
	if _, err := url.Parse(baseURL); err != nil {
		panic(ErrInvalidBaseURL)
	}
}

type ShortenerRequest struct {
	URL string `json:"url"`
}

type ShortenerResponse struct {
	Result string `json:"result"`
}

func (r *router) RedirectURL(c *gin.Context) {
	id := c.Param("hash")

	unShortenURL, err := r.controller.GetURLFromID(c, id)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, unShortenURL.String())
}

func (r *router) CreateShortenerURLRaw(c *gin.Context) {
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
	id, err := r.controller.WriteURL(c, unShortenURL)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	host, _ := url.Parse(r.baseURL)
	host.Path = id
	c.String(http.StatusCreated, host.String())
}

func (r *router) CreateShortenerURLJson(c *gin.Context) {
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
	id, err := r.controller.WriteURL(c, unShortenURL)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	host, _ := url.Parse(r.baseURL)
	host.Path = id
	resp := ShortenerResponse{Result: host.String()}
	c.JSON(http.StatusCreated, resp)
}

func encodingHandler(c *gin.Context) {
	if c.GetHeader("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(c.Request.Body)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Request.Body = gz

		defer gz.Close()
	}
	if c.GetHeader("Accept-Encoding") == "gzip" {
		gz, err := gzip.NewWriterLevel(c.Writer, gzip.BestSpeed)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.Writer = encodeResponseWriter{
			ResponseWriter: c.Writer,
			Writer:         gz,
		}
		defer gz.Close()
		c.Header("Content-Encoding", "gzip")
	}
	c.Next()
}

func errorHandler(c *gin.Context) {
	c.Next()
	for _, e := range c.Errors {
		log.Println(e)
	}
}
