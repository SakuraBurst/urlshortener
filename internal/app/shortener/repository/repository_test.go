package repository

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/url"
	"sync"
	"testing"
	"time"
)

var UnShorterURL = &url.URL{
	Scheme: "http",
	Path:   "test.com",
}

func TestMapBd_ReadFromBd(t *testing.T) {
	type field struct {
		key   string
		value *url.URL
	}
	type preset struct {
		Map    *MapBd
		fields []field
	}
	type args struct {
		ctx context.Context
		id  string
	}
	tests := []struct {
		name         string
		preset       preset
		args         args
		want         *URLTransfer
		positiveTest bool
	}{
		{
			name: "Positive test",
			preset: preset{
				Map:    &MapBd{Map: sync.Map{}},
				fields: []field{{key: "1", value: UnShorterURL}},
			},
			args: args{
				ctx: context.Background(),
				id:  "1",
			},
			want: &URLTransfer{
				UnShorterURL: UnShorterURL,
				Err:          nil,
			},
			positiveTest: true,
		},
		{
			name: "Not found test",
			preset: preset{
				Map:    &MapBd{Map: sync.Map{}},
				fields: []field{},
			},
			args: args{
				ctx: context.Background(),
				id:  "1",
			},
			want: &URLTransfer{
				UnShorterURL: nil,
				Err:          ErrNoSuchURL,
			},
		},
		{
			name: "Context canceled test",
			preset: preset{
				Map:    &MapBd{Map: sync.Map{}},
				fields: []field{},
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				id: "1",
			},
			want: &URLTransfer{
				UnShorterURL: nil,
				Err:          context.Canceled,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, v := range tt.preset.fields {
				tt.preset.Map.Store(v.key, v.value)
			}
			m := tt.preset.Map
			got := m.ReadFromBd(tt.args.ctx, tt.args.id)
			if tt.positiveTest {
				require.NoError(t, got.Err)
			}
			assert.Equal(t, tt.want.UnShorterURL, got.UnShorterURL)

			if !tt.positiveTest {
				assert.Error(t, got.Err)
				assert.ErrorIs(t, got.Err, tt.want.Err)
			}
		})
	}
}

func TestMapBd_WriteToBd(t *testing.T) {
	type preset struct {
		Map *MapBd
	}
	type args struct {
		ctx context.Context
		u   *url.URL
	}
	tests := []struct {
		name         string
		preset       preset
		args         args
		want         *ResultTransfer
		positiveTest bool
	}{
		{
			name:   "Positive test",
			preset: preset{Map: &MapBd{Map: sync.Map{}}},
			args: args{
				ctx: context.Background(),
				u:   UnShorterURL,
			},
			want: &ResultTransfer{
				ID:  "50334",
				Err: nil,
			},
			positiveTest: true,
		},
		{
			name:   "Context cancel test",
			preset: preset{Map: &MapBd{Map: sync.Map{}}},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				u: UnShorterURL,
			},
			want: &ResultTransfer{
				ID:  "",
				Err: context.Canceled,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.preset.Map
			got := m.WriteToBd(tt.args.ctx, tt.args.u)
			if tt.positiveTest {
				require.NoError(t, got.Err)
			}
			assert.Equal(t, tt.want.ID, got.ID)
			if tt.positiveTest {
				u, ok := m.Load(got.ID)
				require.True(t, ok)
				require.Equal(t, tt.args.u, u)
			}

			if !tt.positiveTest {
				assert.Error(t, got.Err)
				assert.ErrorIs(t, got.Err, tt.want.Err)
			}
		})
	}
}

func TestMapBd_getFromBd(t *testing.T) {
	type field struct {
		key   string
		value any
	}
	type preset struct {
		Map    *MapBd
		fields []field
	}
	type args struct {
		ctx     context.Context
		urlChan chan *URLTransfer
		id      string
	}
	tests := []struct {
		name            string
		preset          preset
		args            args
		want            *URLTransfer
		positiveTest    bool
		canceledContext bool
	}{
		{
			name: "Positive test",
			preset: preset{
				Map:    &MapBd{Map: sync.Map{}},
				fields: []field{{key: "1", value: UnShorterURL}},
			},
			args: args{
				ctx:     context.Background(),
				urlChan: make(chan *URLTransfer),
				id:      "1",
			},
			positiveTest: true,
			want: &URLTransfer{
				UnShorterURL: UnShorterURL,
				Err:          nil,
			},
		},
		{
			name: "No url test",
			preset: preset{
				Map: &MapBd{Map: sync.Map{}},
			},
			args: args{
				ctx:     context.Background(),
				urlChan: make(chan *URLTransfer),
				id:      "1",
			},
			want: &URLTransfer{
				UnShorterURL: nil,
				Err:          ErrNoSuchURL,
			},
		},
		{
			name: "Unexpected type in map",
			preset: preset{
				Map:    &MapBd{Map: sync.Map{}},
				fields: []field{{key: "1", value: nil}},
			},
			args: args{
				ctx:     context.Background(),
				urlChan: make(chan *URLTransfer),
				id:      "1",
			},
			want: &URLTransfer{
				UnShorterURL: nil,
				Err:          ErrUnexpectedTypeInMap,
			},
		},
		{
			name: "Canceled",
			preset: preset{
				Map:    &MapBd{Map: sync.Map{}},
				fields: []field{{key: "1", value: nil}},
			},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				urlChan: make(chan *URLTransfer),
			},
			canceledContext: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, v := range tt.preset.fields {
				tt.preset.Map.Store(v.key, v.value)
			}
			m := tt.preset.Map
			go m.getFromBd(tt.args.ctx, tt.args.urlChan, tt.args.id)
			if !tt.canceledContext {
				got := <-tt.args.urlChan
				if tt.positiveTest {
					require.NoError(t, got.Err)
				}
				assert.Equal(t, tt.want.UnShorterURL, got.UnShorterURL)

				if !tt.positiveTest {
					assert.Error(t, got.Err)
					assert.ErrorIs(t, got.Err, tt.want.Err)
				}
			} else {
				time.Sleep(time.Millisecond * 2)
				require.Empty(t, tt.args.urlChan)
			}
			close(tt.args.urlChan)
		})
	}
}

func TestMapBd_writeToBd(t *testing.T) {
	type preset struct {
		Map *MapBd
	}
	type args struct {
		ctx        context.Context
		resultChan chan *ResultTransfer
		u          *url.URL
	}
	tests := []struct {
		name            string
		preset          preset
		args            args
		want            *ResultTransfer
		positiveTest    bool
		canceledContext bool
	}{
		{
			name:   "Positive test",
			preset: preset{Map: &MapBd{Map: sync.Map{}}},
			args: args{
				ctx:        context.Background(),
				resultChan: make(chan *ResultTransfer),
				u:          UnShorterURL,
			},
			want: &ResultTransfer{
				ID:  "50334",
				Err: nil,
			},
			positiveTest: true,
		},
		{
			name:   "Context cancel test",
			preset: preset{Map: &MapBd{Map: sync.Map{}}},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				resultChan: make(chan *ResultTransfer),
				u:          UnShorterURL,
			},
			canceledContext: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.preset.Map
			go m.writeToBd(tt.args.ctx, tt.args.resultChan, tt.args.u)
			if !tt.canceledContext {
				got := <-tt.args.resultChan
				if tt.positiveTest {
					require.NoError(t, got.Err)
				}
				assert.Equal(t, tt.want.ID, got.ID)

				if tt.positiveTest {
					u, ok := m.Load(got.ID)
					require.True(t, ok)
					require.Equal(t, tt.args.u, u)
				}

				if !tt.positiveTest {
					assert.Error(t, got.Err)
					assert.ErrorIs(t, got.Err, tt.want.Err)
				}
			} else {
				time.Sleep(time.Millisecond * 2)
				require.Empty(t, tt.args.resultChan)
			}
			close(tt.args.resultChan)
		})
	}
}
