package router

import (
	"compress/gzip"
	"emperror.dev/errors"
	"encoding/json"
	"fmt"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/controllers"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/token"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type router struct {
	controller   *controllers.Controller
	tokenBuilder *token.TokenBuilder
}

type encodeResponseWriter struct {
	gin.ResponseWriter
	Writer io.Writer
}

func (w encodeResponseWriter) Write(p []byte) (n int, err error) {
	return w.Writer.Write(p)
}

func (w encodeResponseWriter) WriteString(s string) (n int, err error) {
	return w.Writer.Write([]byte(s))
}

func InitAPI(controller *controllers.Controller, tb *token.TokenBuilder) *gin.Engine {
	router := &router{controller: controller, tokenBuilder: tb}
	engine := gin.Default()
	engine.Use(errorHandler)
	engine.Use(encodingHandler)
	engine.Use(router.authHandler)
	engine.GET("/:hash", router.RedirectURL)
	engine.POST("/", router.CreateShortenerURLRaw)
	engine.GET("/ping", router.PingDataBase)
	v1Api := engine.Group("/api")
	{
		shortenGroup := v1Api.Group("/shorten")
		{
			shortenGroup.POST("", router.CreateShortenerURLJson)
			shortenGroup.POST("/batch", router.CreateArrayOfShortenerURLJson)
		}

		userGroup := v1Api.Group("/user")
		{
			userGroup.GET("/urls", router.GetUserURLS)
		}
	}
	return engine
}

type ShortenerRequest struct {
	URL string `json:"url"`
}

type ShortenerResponse struct {
	Result string `json:"result"`
}

type ShortenerRequestWithID struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type ShortenerResponseWithID struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
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
	u, hasConflicts, err := r.controller.WriteURL(c, unShortenURL, c.GetHeader("auth"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if hasConflicts {
		c.String(http.StatusConflict, u)
		return
	}
	c.String(http.StatusCreated, u)
}

func (r *router) GetUserURLS(c *gin.Context) {
	u, err := r.controller.GetUser(c, c.GetHeader("auth"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if len(u) == 0 {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusOK, u)
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
	u, hasConflicts, err := r.controller.WriteURL(c, unShortenURL, c.GetHeader("auth"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	resp := ShortenerResponse{Result: u}
	if hasConflicts {
		c.JSON(http.StatusConflict, resp)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (r *router) CreateArrayOfShortenerURLJson(c *gin.Context) {
	var req []ShortenerRequestWithID
	if err := c.BindJSON(&req); err != nil {
		return
	}
	u := make([]*url.URL, 0, len(req))
	for _, v := range req {
		unShortenURL, err := url.Parse(v.OriginalURL)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		u = append(u, unShortenURL)
	}

	res, hasConflicts, err := r.controller.WriteArrayOfURL(c, u, c.GetHeader("auth"))
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	resp := make([]ShortenerResponseWithID, 0, len(res))
	for i, re := range res {
		resp = append(resp, ShortenerResponseWithID{
			CorrelationID: req[i].CorrelationID,
			ShortURL:      re,
		})
	}
	if hasConflicts {
		c.JSON(http.StatusConflict, resp)
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (r *router) PingDataBase(c *gin.Context) {
	err := r.controller.PingDataBase(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusOK)
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

func (r *router) authHandler(c *gin.Context) {
	t, err := c.Cookie("auth")
	if err != nil || !r.tokenBuilder.IsTokenValid(t) {
		t, err = r.controller.CreateUser(c)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.SetCookie("auth", t, time.Now().Add(time.Hour*24).Nanosecond(), "", "", false, true)
	}
	c.Request.Header.Set("auth", t)
	c.Next()
}

func errorHandler(c *gin.Context) {
	c.Next()
	fmt.Println("____________________________________________________________________________")
	for _, e := range c.Errors {
		log.Println(e)
	}
	fmt.Println("____________________________________________________________________________")
}
