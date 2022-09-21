package router

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/controllers"
	"github.com/SakuraBurst/urlshortener/internal/app/shortener/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const localhost = "http://localhost:8080"

func createRequest(t *testing.T, method string, url string, body io.Reader) *http.Request {
	request, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	return request
}

type mockDataBase struct {
	mock.Mock
}

func (r *mockDataBase) Read(ctx context.Context, s string) (any, error) {
	args := r.Called(s)
	return args.Get(0), args.Error(1)
}

func (r *mockDataBase) Create(ctx context.Context, val any) (string, error) {
	args := r.Called(val)
	return args.String(0), args.Error(1)
}

func (r *mockDataBase) CreateArray(ctx context.Context, val any) ([]string, error) {
	panic("not implemented")
}

func (r *mockDataBase) Update(ctx context.Context, id string, val any) error {
	args := r.Called(id, val)
	return args.Error(0)
}

var MockURLRaw = "https://test.com/"

var hashURL = "1"

var MockURLShorten = localhost + "/" + hashURL

var MockURL, _ = url.Parse(MockURLRaw)

func TestCreateShortenerURLRaw(t *testing.T) {
	type want struct {
		statusCode  int
		contentType string
	}
	type args struct {
		writer  *httptest.ResponseRecorder
		request *http.Request
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
				request: createRequest(t, http.MethodPost, "/", bytes.NewBuffer([]byte(MockURLRaw))),
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
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
			},
			positiveTest: false,
		},
	}
	urlDB := new(mockDataBase)
	userDB := new(mockDataBase)
	urlDB.On("Create", MockURL).Return("1", nil).Once()
	userDB.On("Create", []string(nil)).Return("1", nil).Times(len(tests))
	userDB.On("Read", "1").Return([]string(nil), nil).Once()
	userDB.On("Update", "1", []string{hashURL}).Return(nil).Once()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := InitAPI(controllers.InitController(localhost, nil, urlDB, userDB))
			router.ServeHTTP(tt.args.writer, tt.args.request)
			result := tt.args.writer.Result()
			assert.Equal(t, tt.want.contentType, result.Header.Get("content-type"))
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			buf := bytes.NewBuffer(nil)
			defer result.Body.Close()
			_, err := io.Copy(buf, result.Body)
			require.NoError(t, err)
		})
	}
	urlDB.AssertExpectations(t)
	userDB.AssertExpectations(t)
}

func TestCreateShortenerURLJson(t *testing.T) {
	type want struct {
		statusCode  int
		contentType string
	}
	type args struct {
		writer  *httptest.ResponseRecorder
		request *http.Request
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
				request: createRequest(t, http.MethodPost, "/api/shorten", bytes.NewBuffer([]byte(fmt.Sprintf(`{"url": "%s"}`, MockURLRaw)))),
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
			},
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: "",
			},
			positiveTest: false,
		},
	}
	urlDB := new(mockDataBase)
	userDB := new(mockDataBase)
	urlDB.On("Create", MockURL).Return("1", nil).Once()
	userDB.On("Create", []string(nil)).Return("1", nil).Times(len(tests))
	userDB.On("Read", "1").Return([]string(nil), nil).Once()
	userDB.On("Update", "1", []string{hashURL}).Return(nil).Once()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := InitAPI(controllers.InitController(localhost, nil, urlDB, userDB))
			router.ServeHTTP(tt.args.writer, tt.args.request)
			result := tt.args.writer.Result()
			assert.Equal(t, tt.want.contentType, result.Header.Get("content-type"))
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			buf := bytes.NewBuffer(nil)
			defer result.Body.Close()
			_, err := io.Copy(buf, result.Body)
			require.NoError(t, err)
		})
	}
	urlDB.AssertExpectations(t)
	userDB.AssertExpectations(t)
}

func TestRedirectURL(t *testing.T) {
	type want struct {
		statusCode int
		location   string
	}
	type args struct {
		writer  *httptest.ResponseRecorder
		request *http.Request
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
			},
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				location:   MockURLRaw,
			},
			positiveTest: true,
		},
		{
			name: "no such url test",
			args: args{
				writer:  httptest.NewRecorder(),
				request: createRequest(t, http.MethodGet, "/2", bytes.NewBuffer([]byte{0})),
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
			},
			want: want{
				statusCode: http.StatusNotFound,
			},
			positiveTest: false,
		},
	}
	urlDB := new(mockDataBase)
	userDB := new(mockDataBase)
	urlDB.On("Read", "1").Return(MockURL, nil).Once()
	urlDB.On("Read", "2").Return(nil, repository.ErrNoSuchValue).Once()
	userDB.On("Create", []string(nil)).Return("1", nil).Times(len(tests))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := InitAPI(controllers.InitController(localhost, nil, urlDB, userDB))
			router.ServeHTTP(tt.args.writer, tt.args.request)
			result := tt.args.writer.Result()
			result.Body.Close()
			assert.Equal(t, tt.want.statusCode, result.StatusCode)
			if tt.positiveTest {
				assert.Equal(t, tt.want.location, result.Header.Get("Location"))
			}
		})
	}
	urlDB.AssertExpectations(t)
	userDB.AssertExpectations(t)
}

func TestNotFoundEndpoint(t *testing.T) {
	DB := new(mockDataBase)
	DB.On("Create", []string(nil)).Return("1", nil).Once()
	router := InitAPI(controllers.InitController(localhost, nil, DB, DB))
	request := createRequest(t, http.MethodPost, "/asdfalfkasdfkkjasdfasfasfasdfsaf", bytes.NewBuffer([]byte{0}))
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
					payload:       MockURLRaw,
					needToEncode:  true,
					setGzipHeader: true,
				},
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
					payload:       MockURLRaw,
					needToEncode:  false,
					setGzipHeader: true,
				},
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
					payload: MockURLRaw,
				},
			},
			want: want{
				statusCode:      http.StatusCreated,
				contentEncoding: "gzip",
				decodedString:   MockURLShorten,
			},
			positiveTest: true,
			encodingTest: true,
		},
	}
	urlDB := new(mockDataBase)
	userDB := new(mockDataBase)
	urlDB.On("Create", MockURL).Return("1", nil).Twice()
	userDB.On("Create", []string(nil)).Return("1", nil).Times(len(tests) - 1)
	userDB.On("Read", "1").Return([]string(nil), nil).Twice()
	userDB.On("Update", "1", []string{hashURL}).Return(nil).Twice()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := InitAPI(controllers.InitController(localhost, nil, urlDB, userDB))
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
				if tt.encodingTest {
					r, err := gzip.NewReader(buf)
					require.NoError(t, err)
					b, err := io.ReadAll(r)
					require.NoError(t, err)
					assert.Equal(t, tt.want.decodedString, string(b))
				}
			}
		})
	}
	urlDB.AssertExpectations(t)
	userDB.AssertExpectations(t)
}
