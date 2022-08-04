package api

import (
	"bytes"
	"context"
	"github.com/SakuraBurst/urlshortener/internal/controlers"
	"github.com/SakuraBurst/urlshortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func createRequest(t *testing.T, method string, url string, body io.Reader) *http.Request {
	request, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	return request
}

type repo map[string]*url.URL

func (r repo) Read(ctx context.Context, s string) *repository.URLTransfer {
	if _, ok := r[s]; !ok {
		return &repository.URLTransfer{
			UnShorterURL: nil,
			Err:          repository.NoSuchURLError,
		}
	}
	return &repository.URLTransfer{
		UnShorterURL: r[s],
		Err:          nil,
	}
}

func (r repo) Write(ctx context.Context, url *url.URL) *repository.ResultTransfer {
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

func TestCreateShortenerURL(t *testing.T) {
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
				request: createRequest(t, http.MethodPost, "/", bytes.NewBuffer([]byte("https://vk.com/feed"))),
				bd:      repo{},
			},
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
			},
			positiveTest: true,
		},
		{
			name: "bad url test",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodPost, "/", bytes.NewBuffer([]byte{0})),
				bd:      repo{},
			},
			want: want{
				statusCode:  http.StatusInternalServerError,
				contentType: "",
			},
			positiveTest: false,
		},
		{
			name: "nil body",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodPost, "/", bytes.NewBuffer(nil)),
				bd:      repo{},
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
			controlers.SetRepository(tt.args.bd)
			CreateShortenerURL(tt.args.writer, tt.args.request)
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

func TestInitAPI(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitAPI()
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
				request: createRequest(t, http.MethodGet, "/?id=1", nil),
				bd: repo{"1": &url.URL{
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
				request: createRequest(t, http.MethodGet, "/?id=1", bytes.NewBuffer([]byte{0})),
				bd:      repo{},
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
				bd:      repo{},
			},
			want: want{
				statusCode: http.StatusBadRequest,
			},
			positiveTest: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlers.SetRepository(tt.args.bd)
			RedirectURL(tt.args.writer, tt.args.request)
			result := tt.args.writer.Result()
			result.Body.Close()
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			if tt.positiveTest {
				assert.Equal(t, tt.want.location, result.Header.Get("Location"))
			}
		})
	}
}

func Test_errorHandler(t *testing.T) {
	type args struct {
		writer     http.ResponseWriter
		statusCode int
		err        error
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errorHandler(tt.args.writer, tt.args.statusCode, tt.args.err)
		})
	}
}
