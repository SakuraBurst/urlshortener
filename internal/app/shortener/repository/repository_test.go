package repository

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
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
		Map    *SyncMapRepo
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
				Map:    &SyncMapRepo{},
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
				Map:    &SyncMapRepo{},
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
				Map:    &SyncMapRepo{},
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
				tt.preset.Map.sMap.Store(v.key, v.value)
			}
			m := tt.preset.Map
			res, err := m.Read(tt.args.ctx, tt.args.id)
			if tt.positiveTest {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want.UnShorterURL, res)

			if !tt.positiveTest {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.want.Err)
			}
		})
	}
}

func TestMapBd_WriteToBd(t *testing.T) {
	type preset struct {
		Map *SyncMapRepo
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
			preset: preset{Map: &SyncMapRepo{}},
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
			preset: preset{Map: &SyncMapRepo{}},
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
			res, err := m.Write(tt.args.ctx, tt.args.u)
			if tt.positiveTest {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want.ID, res)
			if tt.positiveTest {
				u, ok := m.sMap.Load(res)
				require.True(t, ok)
				require.Equal(t, tt.args.u, u)
			}

			if !tt.positiveTest {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.want.Err)
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
		Map    *SyncMapRepo
		fields []field
	}
	type args struct {
		urlChan chan *URLTransfer
		id      string
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
				Map:    &SyncMapRepo{},
				fields: []field{{key: "1", value: UnShorterURL}},
			},
			args: args{
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
				Map: &SyncMapRepo{},
			},
			args: args{
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
				Map:    &SyncMapRepo{},
				fields: []field{{key: "1", value: nil}},
			},
			args: args{
				urlChan: make(chan *URLTransfer),
				id:      "1",
			},
			want: &URLTransfer{
				UnShorterURL: nil,
				Err:          ErrUnexpectedTypeInMap,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, v := range tt.preset.fields {
				tt.preset.Map.sMap.Store(v.key, v.value)
			}
			m := tt.preset.Map
			go m.getFromDB(tt.args.urlChan, tt.args.id)
			got := <-tt.args.urlChan
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

func TestMapBd_writeToBd(t *testing.T) {
	type preset struct {
		Map *SyncMapRepo
	}
	type args struct {
		resultChan chan *ResultTransfer
		u          *url.URL
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
			preset: preset{Map: &SyncMapRepo{}},
			args: args{
				resultChan: make(chan *ResultTransfer),
				u:          UnShorterURL,
			},
			want: &ResultTransfer{
				ID:  "50334",
				Err: nil,
			},
			positiveTest: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.preset.Map
			go m.writeToDB(tt.args.resultChan, tt.args.u)
			got := <-tt.args.resultChan
			if tt.positiveTest {
				require.NoError(t, got.Err)
			}
			assert.Equal(t, tt.want.ID, got.ID)

			if tt.positiveTest {
				u, ok := m.sMap.Load(got.ID)
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
