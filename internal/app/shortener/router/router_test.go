package router

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/controllers"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/types"
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

type mockURLDataBase map[string]*url.URL

func (r mockURLDataBase) Read(ctx context.Context, s string) (any, error) {
	if _, ok := r[s]; !ok {
		return nil, repository.ErrNoSuchValue
	}
	return r[s], nil
}

func (r mockURLDataBase) Create(ctx context.Context, val any) (string, error) {
	u, ok := val.(*url.URL)

	if !ok {
		return "", fmt.Errorf("URL repository dont support this type of value - %T", val)
	}
	builder := strings.Builder{}
	builder.WriteString("a")
	for _, ok := r[builder.String()]; ok; {
		builder.WriteString("a")
	}
	r[builder.String()] = u
	return builder.String(), nil
}

func (r mockURLDataBase) Update(ctx context.Context, id string, val any) error {
	u, ok := val.(*url.URL)
	if !ok {
		return fmt.Errorf("URL repository dont support this type of value - %T", val)
	}
	r[id] = u
	return nil
}

type mockUserDataBase map[string][]*types.URLShorter

func (r mockUserDataBase) Read(ctx context.Context, s string) (any, error) {
	if _, ok := r[s]; !ok {
		return nil, repository.ErrNoSuchValue
	}
	return r[s], nil
}

func (r mockUserDataBase) Create(ctx context.Context, val any) (string, error) {
	u, ok := val.([]*types.URLShorter)
	if !ok {
		return "", fmt.Errorf("USER repository dont support this type of value - %T", val)
	}
	builder := strings.Builder{}
	builder.WriteString("1")
	for _, ok := r[builder.String()]; ok; {
		builder.WriteString("1")
	}
	r[builder.String()] = u
	return builder.String(), nil
}

func (r mockUserDataBase) Update(ctx context.Context, id string, val any) error {
	u, ok := val.([]*types.URLShorter)
	if !ok {
		return fmt.Errorf("USER repository dont support this type of value - %T", val)
	}
	r[id] = u
	return nil
}

func TestCreateShortenerURLRaw(t *testing.T) {
	type want struct {
		statusCode  int
		contentType string
	}
	type args struct {
		writer  *httptest.ResponseRecorder
		request *http.Request
		db      repository.Repository
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
				db:      mockURLDataBase{},
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
				db:      mockURLDataBase{},
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
				db:      mockURLDataBase{},
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
			},
			positiveTest: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := InitAPI(controllers.InitController(localhost, tt.args.db, mockUserDataBase{}))
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
				assert.NotEmpty(t, tt.args.db)
			} else {
				assert.Empty(t, tt.args.db)
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
		db      repository.Repository
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
				db:      mockURLDataBase{},
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
				db:      mockURLDataBase{},
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
				db:      mockURLDataBase{},
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
			},
			positiveTest: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := InitAPI(controllers.InitController(localhost, tt.args.db, mockUserDataBase{}))
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
				assert.NotEmpty(t, tt.args.db)
			} else {
				assert.Empty(t, tt.args.db)
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
		db      repository.Repository
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
				db: mockURLDataBase{"1": &url.URL{
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
				db:      mockURLDataBase{},
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
				db:      mockURLDataBase{},
			},
			want: want{
				statusCode: http.StatusNotFound,
			},
			positiveTest: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := InitAPI(controllers.InitController(localhost, tt.args.db, mockUserDataBase{}))
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
	router := InitAPI(controllers.InitController(localhost, mockURLDataBase{}, mockUserDataBase{}))
	request := createRequest(t, http.MethodGet, "/asdfalfkasdfkkjasdfasfasfasdfsaf", bytes.NewBuffer([]byte{0}))
	writer := httptest.NewRecorder()
	router.ServeHTTP(writer, request)
	result := writer.Result()
	result.Body.Close()
	assert.Equal(t, http.StatusNotFound, result.StatusCode)
}

//func Test_checkBaseURL(t *testing.T) {
//	type args struct {
//		baseURL string
//	}
//	tests := []struct {
//		name       string
//		args       args
//		err        error
//		isPositive bool
//	}{
//		{
//			name:       "Positive test",
//			args:       args{baseURL: localhost},
//			isPositive: true,
//		},
//		{
//			name:       "NoUrl",
//			args:       args{},
//			isPositive: false,
//			err:        ErrNoBaseURL,
//		},
//		{
//			name:       "BadUrl",
//			args:       args{baseURL: string([]byte{0})},
//			isPositive: false,
//			err:        ErrInvalidBaseURL,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if tt.isPositive {
//				require.NotPanics(t, func() {
//					checkBaseURL(tt.args.baseURL)
//				})
//			} else {
//				require.PanicsWithError(t, tt.err.Error(), func() {
//					checkBaseURL(tt.args.baseURL)
//				})
//			}
//		})
//	}
//}

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
		db      repository.Repository
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
				db: mockURLDataBase{},
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
				db: mockURLDataBase{},
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
				db: mockURLDataBase{},
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := InitAPI(controllers.InitController(localhost, tt.args.db, mockUserDataBase{}))
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
				assert.NotEmpty(t, tt.args.db)
				if tt.encodingTest {
					r, err := gzip.NewReader(buf)
					require.NoError(t, err)
					b, err := io.ReadAll(r)
					require.NoError(t, err)
					assert.Equal(t, tt.want.decodedString, string(b))
				}
			} else {
				assert.Empty(t, tt.args.db)
			}
		})
	}
}
