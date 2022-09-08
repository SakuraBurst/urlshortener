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
		Map    *SyncMapURLRepo
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
		want         *valueTransfer
		positiveTest bool
	}{
		{
			name: "Positive test",
			preset: preset{
				Map:    &SyncMapURLRepo{},
				fields: []field{{key: "1", value: UnShorterURL}},
			},
			args: args{
				ctx: context.Background(),
				id:  "1",
			},
			want: &valueTransfer{
				value: UnShorterURL,
				err:   nil,
			},
			positiveTest: true,
		},
		{
			name: "Not found test",
			preset: preset{
				Map:    &SyncMapURLRepo{},
				fields: []field{},
			},
			args: args{
				ctx: context.Background(),
				id:  "1",
			},
			want: &valueTransfer{
				value: nil,
				err:   ErrNoSuchValue,
			},
		},
		{
			name: "Context canceled test",
			preset: preset{
				Map:    &SyncMapURLRepo{},
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
			want: &valueTransfer{
				value: nil,
				err:   context.Canceled,
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
			assert.Equal(t, tt.want.value, res)

			if !tt.positiveTest {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.want.err)
			}
		})
	}
}

func TestMapBd_WriteToBd(t *testing.T) {
	type preset struct {
		Map *SyncMapURLRepo
	}
	type args struct {
		ctx context.Context
		u   *url.URL
	}
	tests := []struct {
		name         string
		preset       preset
		args         args
		want         *resultIdTransfer
		positiveTest bool
	}{
		{
			name:   "Positive test",
			preset: preset{Map: &SyncMapURLRepo{}},
			args: args{
				ctx: context.Background(),
				u:   UnShorterURL,
			},
			want: &resultIdTransfer{
				id:  "50334",
				err: nil,
			},
			positiveTest: true,
		},
		{
			name:   "Context cancel test",
			preset: preset{Map: &SyncMapURLRepo{}},
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				u: UnShorterURL,
			},
			want: &resultIdTransfer{
				id:  "",
				err: context.Canceled,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.preset.Map
			res, err := m.Create(tt.args.ctx, tt.args.u)
			if tt.positiveTest {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want.id, res)
			if tt.positiveTest {
				u, ok := m.sMap.Load(res)
				require.True(t, ok)
				require.Equal(t, tt.args.u, u)
			}

			if !tt.positiveTest {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.want.err)
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
		Map    *SyncMapURLRepo
		fields []field
	}
	type args struct {
		urlChan chan *valueTransfer
		id      string
	}
	tests := []struct {
		name         string
		preset       preset
		args         args
		want         *valueTransfer
		positiveTest bool
	}{
		{
			name: "Positive test",
			preset: preset{
				Map:    &SyncMapURLRepo{},
				fields: []field{{key: "1", value: UnShorterURL}},
			},
			args: args{
				urlChan: make(chan *valueTransfer),
				id:      "1",
			},
			positiveTest: true,
			want: &valueTransfer{
				value: UnShorterURL,
				err:   nil,
			},
		},
		{
			name: "No url test",
			preset: preset{
				Map: &SyncMapURLRepo{},
			},
			args: args{
				urlChan: make(chan *valueTransfer),
				id:      "1",
			},
			want: &valueTransfer{
				value: nil,
				err:   ErrNoSuchValue,
			},
		},
		{
			name: "Unexpected type in map",
			preset: preset{
				Map:    &SyncMapURLRepo{},
				fields: []field{{key: "1", value: nil}},
			},
			args: args{
				urlChan: make(chan *valueTransfer),
				id:      "1",
			},
			want: &valueTransfer{
				value: nil,
				err:   ErrUnexpectedTypeInMap,
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
				require.NoError(t, got.err)
			}
			assert.Equal(t, tt.want.value, got.value)

			if !tt.positiveTest {
				assert.Error(t, got.err)
				assert.ErrorIs(t, got.err, tt.want.err)
			}

		})
	}
}

func TestMapBd_writeToBd(t *testing.T) {
	type preset struct {
		Map *SyncMapURLRepo
	}
	type args struct {
		resultChan chan *resultIdTransfer
		u          *url.URL
	}
	tests := []struct {
		name         string
		preset       preset
		args         args
		want         *resultIdTransfer
		positiveTest bool
	}{
		{
			name:   "Positive test",
			preset: preset{Map: &SyncMapURLRepo{}},
			args: args{
				resultChan: make(chan *resultIdTransfer),
				u:          UnShorterURL,
			},
			want: &resultIdTransfer{
				id:  "50334",
				err: nil,
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
				require.NoError(t, got.err)
			}
			assert.Equal(t, tt.want.id, got.id)

			if tt.positiveTest {
				u, ok := m.sMap.Load(got.id)
				require.True(t, ok)
				require.Equal(t, tt.args.u, u)
			}

			if !tt.positiveTest {
				assert.Error(t, got.err)
				assert.ErrorIs(t, got.err, tt.want.err)
			}

		})
	}
}
