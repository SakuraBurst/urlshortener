package router

import (
	"bytes"
	"compress/gzip"
	"context"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/controlers"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const localhost = "http://localhost:8080"

func createRequest(t *testing.T, method string, url string, body io.Reader) *http.Request {
	request, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	return request
}

type testBd map[string]*url.URL

func (r testBd) ReadFromBd(ctx context.Context, s string) *repository.URLTransfer {
	if _, ok := r[s]; !ok {
		return &repository.URLTransfer{
			UnShorterURL: nil,
			Err:          repository.ErrNoSuchURL,
		}
	}
	return &repository.URLTransfer{
		UnShorterURL: r[s],
		Err:          nil,
	}
}

func (r testBd) WriteToBd(ctx context.Context, url *url.URL) *repository.ResultTransfer {
	builder := strings.Builder{}
	builder.WriteString("a")
	for _, ok := r[builder.String()]; ok; {
		builder.WriteString("a")
	}
	r[builder.String()] = url
	return &repository.ResultTransfer{
		ID:  builder.String(),
		Err: nil,
	}
}
func (r testBd) InitRepository(key string) {
	panic("InitRepository is unsupported for testBd")
}

func TestCreateShortenerURLRaw(t *testing.T) {
	type want struct {
		statusCode  int
		contentType string
	}
	type args struct {
		writer  *httptest.ResponseRecorder
		request *http.Request
		bd      repository.URLShortenerRepository
	}
	tests := []struct {
		name         string
		args         args
		want         want
		positiveTest bool
	}{
		{
			name: "positive test",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodPost, "/", bytes.NewBuffer([]byte("https://test.com/"))),
				bd:      testBd{},
			},
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain; charset=utf-8",
			},
			positiveTest: true,
		},
		{
			name: "bad url test",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodPost, "/", bytes.NewBuffer([]byte{0})),
				bd:      testBd{},
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
			},
			positiveTest: false,
		},
		{
			name: "nil body",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodPost, "/", bytes.NewBuffer(nil)),
				bd:      testBd{},
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
			},
			positiveTest: false,
		},
	}
	router := InitAPI(localhost)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlers.SetRepository(tt.args.bd)
			router.ServeHTTP(tt.args.writer, tt.args.request)
			result := tt.args.writer.Result()
			assert.Equal(t, tt.want.contentType, result.Header.Get("content-type"))
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			buf := bytes.NewBuffer(nil)
			defer result.Body.Close()
			_, err := io.Copy(buf, result.Body)
			require.NoError(t, err)
			if tt.positiveTest {
				assert.NotEmpty(t, buf.Bytes())
				assert.NotEmpty(t, tt.args.bd)
			} else {
				assert.Empty(t, tt.args.bd)
			}
		})
	}
}

func TestCreateShortenerURLJson(t *testing.T) {
	type want struct {
		statusCode  int
		contentType string
	}
	type args struct {
		writer  *httptest.ResponseRecorder
		request *http.Request
		bd      repository.URLShortenerRepository
	}
	tests := []struct {
		name         string
		args         args
		want         want
		positiveTest bool
	}{
		{
			name: "positive test",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodPost, "/api/shorten", bytes.NewBuffer([]byte(`{"url": "https://test.com/"}`))),
				bd:      testBd{},
			},
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "application/json; charset=utf-8",
			},
			positiveTest: true,
		},
		{
			name: "bad url test",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodPost, "/api/shorten", bytes.NewBuffer([]byte(`{"url": "\n"}`))),
				bd:      testBd{},
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
			},
			positiveTest: false,
		},
		{
			name: "nil body",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodPost, "/api/shorten", bytes.NewBuffer(nil)),
				bd:      testBd{},
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
			},
			positiveTest: false,
		},
	}
	router := InitAPI(localhost)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlers.SetRepository(tt.args.bd)
			router.ServeHTTP(tt.args.writer, tt.args.request)
			result := tt.args.writer.Result()
			assert.Equal(t, tt.want.contentType, result.Header.Get("content-type"))
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			buf := bytes.NewBuffer(nil)
			defer result.Body.Close()
			_, err := io.Copy(buf, result.Body)
			require.NoError(t, err)
			if tt.positiveTest {
				assert.NotEmpty(t, buf.Bytes())
				assert.NotEmpty(t, tt.args.bd)
			} else {
				assert.Empty(t, tt.args.bd)
			}
		})
	}
}

func TestRedirectURL(t *testing.T) {
	type want struct {
		statusCode int
		location   string
	}
	type args struct {
		writer  *httptest.ResponseRecorder
		request *http.Request
		bd      repository.URLShortenerRepository
	}
	tests := []struct {
		name         string
		args         args
		want         want
		positiveTest bool
	}{
		{
			name: "positive test",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodGet, "/1", nil),
				bd: testBd{"1": &url.URL{
					Scheme: "https",
					Host:   "www.google.com",
					Path:   "/",
				}},
			},
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				location:   "https://www.google.com/",
			},
			positiveTest: true,
		},
		{
			name: "no such url test",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodGet, "/1", bytes.NewBuffer([]byte{0})),
				bd:      testBd{},
			},
			want: want{
				statusCode: http.StatusNotFound,
			},
			positiveTest: false,
		},
		{
			name: "no id test",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodGet, "/", bytes.NewBuffer(nil)),
				bd:      testBd{},
			},
			want: want{
				statusCode: http.StatusNotFound,
			},
			positiveTest: false,
		},
	}
	router := InitAPI(localhost)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlers.SetRepository(tt.args.bd)
			router.ServeHTTP(tt.args.writer, tt.args.request)
			result := tt.args.writer.Result()
			result.Body.Close()
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			if tt.positiveTest {
				assert.Equal(t, tt.want.location, result.Header.Get("Location"))
			}
		})
	}
}

func TestNotFoundEndpoint(t *testing.T) {
	router := InitAPI(localhost)
	request := createRequest(t, http.MethodGet, "/asdfalfkasdfkkjasdfasfasfasdfsaf", bytes.NewBuffer([]byte{0}))
	writer := httptest.NewRecorder()
	router.ServeHTTP(writer, request)
	result := writer.Result()
	result.Body.Close()
	assert.Equal(t, http.StatusNotFound, result.StatusCode)
}

func Test_checkBaseURL(t *testing.T) {
	type args struct {
		baseURL string
	}
	tests := []struct {
		name       string
		args       args
		err        error
		isPositive bool
	}{
		{
			name:       "Positive test",
			args:       args{baseURL: localhost},
			isPositive: true,
		},
		{
			name:       "NoUrl",
			args:       args{},
			isPositive: false,
			err:        ErrNoBaseURL,
		},
		{
			name:       "BadUrl",
			args:       args{baseURL: string([]byte{0})},
			isPositive: false,
			err:        ErrInvalidBaseURL,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.isPositive {
				require.NotPanics(t, func() {
					checkBaseURL(tt.args.baseURL)
				})
			} else {
				require.PanicsWithError(t, tt.err.Error(), func() {
					checkBaseURL(tt.args.baseURL)
				})
			}
		})
	}
}

func Test_encodingHandler(t *testing.T) {
	type want struct {
		statusCode      int
		contentEncoding string
		decodedString   string
	}
	type request struct {
		method        string
		route         string
		payload       string
		needToEncode  bool
		setGzipHeader bool
	}
	type args struct {
		writer  *httptest.ResponseRecorder
		request request
		bd      repository.URLShortenerRepository
	}
	tests := []struct {
		name         string
		args         args
		want         want
		positiveTest bool
		encodingTest bool
	}{
		{
			name: "encoded payload test",
			args: args{
				writer: httptest.NewRecorder(),
				request: request{
					method:        http.MethodPost,
					route:         "/",
					payload:       "https://test.com/",
					needToEncode:  true,
					setGzipHeader: true,
				},
				bd: testBd{},
			},
			want: want{
				statusCode: http.StatusCreated,
			},
			positiveTest: true,
		},
		{
			name: "unencoded payload test",
			args: args{
				writer: httptest.NewRecorder(),
				request: request{
					method:        http.MethodPost,
					route:         "/",
					payload:       "https://test.com/",
					needToEncode:  false,
					setGzipHeader: true,
				},
				bd: testBd{},
			},
			want: want{
				statusCode: http.StatusInternalServerError,
			},
			positiveTest: false,
		},
		{
			name: "encoded response test",
			args: args{
				writer: httptest.NewRecorder(),
				request: request{
					method:  http.MethodPost,
					route:   "/",
					payload: "https://test.com/",
				},
				bd: testBd{},
			},
			want: want{
				statusCode:      http.StatusCreated,
				contentEncoding: "gzip",
				decodedString:   "http://localhost:8080/a",
			},
			positiveTest: true,
			encodingTest: true,
		},
	}
	router := InitAPI(localhost)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlers.SetRepository(tt.args.bd)
			b := bytes.NewBuffer(nil)
			if tt.args.request.needToEncode {
				w := gzip.NewWriter(b)
				_, err := w.Write([]byte(tt.args.request.payload))
				require.NoError(t, err)
				err = w.Close()
				require.NoError(t, err)
			} else {
				_, err := b.WriteString(tt.args.request.payload)
				require.NoError(t, err)
			}
			req := createRequest(t, tt.args.request.method, tt.args.request.route, b)
			if tt.args.request.setGzipHeader {
				req.Header.Set("Content-Encoding", "gzip")
			}
			if tt.encodingTest {
				req.Header.Set("Accept-Encoding", "gzip")
			}
			router.ServeHTTP(tt.args.writer, req)
			result := tt.args.writer.Result()
			assert.Equal(t, tt.want.contentEncoding, result.Header.Get("content-encoding"))
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			buf := bytes.NewBuffer(nil)
			defer result.Body.Close()
			_, err := io.Copy(buf, result.Body)
			require.NoError(t, err)
			if tt.positiveTest {
				assert.NotEmpty(t, buf.Bytes())
				assert.NotEmpty(t, tt.args.bd)
				if tt.encodingTest {
					r, err := gzip.NewReader(buf)
					require.NoError(t, err)
					b, err := io.ReadAll(r)
					require.NoError(t, err)
					assert.Equal(t, tt.want.decodedString, string(b))
				}
			} else {
				assert.Empty(t, tt.args.bd)
			}
		})
	}
}
