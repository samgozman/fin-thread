package jobs

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/samgozman/fin-thread/archivist"
	"github.com/samgozman/fin-thread/composer"
	"github.com/samgozman/fin-thread/scavenger/stocks"
	"reflect"
	"testing"
)

func Test_formatNewsWithComposedMeta(t *testing.T) {
	type args struct {
		n archivist.News
	}
	d1, _ := json.Marshal(composer.ComposedMeta{
		Tickers: []string{"AAPL"},
	})
	d2, _ := json.Marshal(composer.ComposedMeta{
		Tickers: []string{"AAPL", "MSFT"},
	})
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test1",
			args: args{
				n: archivist.News{
					ID:           uuid.New(),
					ComposedText: "Some AAPL news about AAPL stock.",
					MetaData:     d1,
				},
			},
			want: "Some [AAPL](https://short-fork.extr.app/en/AAPL?utm_source=finthread) news about AAPL stock.",
		},
		{
			name: "test2",
			args: args{
				n: archivist.News{
					ID:           uuid.New(),
					ComposedText: "Some N1N2N3 news about some stock.",
					MetaData:     nil,
				},
			},
			want: "Some N1N2N3 news about some stock.",
		},
		{
			name: "multiple tickers",
			args: args{
				n: archivist.News{
					ID:           uuid.New(),
					ComposedText: "Some AAPL news about with MSFT stock.",
					MetaData:     d2,
				},
			},
			want: "Some [AAPL](https://short-fork.extr.app/en/AAPL?utm_source=finthread) news about with [MSFT](https://short-fork.extr.app/en/MSFT?utm_source=finthread) stock.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatNewsWithComposedMeta(tt.args.n); got != tt.want {
				t.Errorf("formatNewsWithComposedMeta() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJob_prepublishFilter(t *testing.T) {
	type fields struct {
		stocks  *stocks.StockMap
		options *jobOptions
	}
	type args struct {
		news []*archivist.News
	}

	d1, _ := json.Marshal(composer.ComposedMeta{
		Tickers: []string{"AAPL"},
	})
	d2, _ := json.Marshal(composer.ComposedMeta{
		Tickers: []string{"PLTR"},
	})
	emptyMeta, _ := json.Marshal(composer.ComposedMeta{})

	okID := uuid.New()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*archivist.News
		wantErr bool
	}{
		{
			name: "No filters",
			fields: fields{
				stocks:  nil,
				options: &jobOptions{},
			},
			args: args{
				news: []*archivist.News{},
			},
			want:    []*archivist.News{},
			wantErr: false,
		},
		{
			name: "Omit suspicious news",
			fields: fields{
				stocks: nil,
				options: &jobOptions{
					omitSuspicious: true,
				},
			},
			args: args{
				news: []*archivist.News{
					{
						ID:           uuid.New(),
						ComposedText: "Some AAPL news about AAPL stock.",
						MetaData:     d1,
						IsSuspicious: true,
					},
					{
						ID:           okID,
						ComposedText: "Some other AAPL news.",
						MetaData:     d1,
						IsSuspicious: false,
					},
				},
			},
			want: []*archivist.News{
				{
					ID:           okID,
					ComposedText: "Some other AAPL news.",
					MetaData:     d1,
					IsSuspicious: false,
				},
			},
			wantErr: false,
		},
		{
			name: "Omit news with empty tickers",
			fields: fields{
				stocks: nil,
				options: &jobOptions{
					omitEmptyMetaKeys: &omitKeyOptions{
						emptyTickers: true,
					},
				},
			},
			args: args{
				news: []*archivist.News{
					{
						ID:           uuid.New(),
						ComposedText: "Some AAPL news without meta.",
						MetaData:     emptyMeta,
						IsSuspicious: false,
					},
					{
						ID:           okID,
						ComposedText: "Some other AAPL news.",
						MetaData:     d1,
						IsSuspicious: false,
					},
				},
			},
			want: []*archivist.News{
				{
					ID:           okID,
					ComposedText: "Some other AAPL news.",
					MetaData:     d1,
					IsSuspicious: false,
				},
			},
			wantErr: false,
		},
		{
			name: "Omit unlisted stocks",
			fields: fields{
				stocks: &stocks.StockMap{
					"AAPL": stocks.Stock{},
				},
				options: &jobOptions{
					omitUnlistedStocks: true,
				},
			},
			args: args{
				news: []*archivist.News{
					{
						ID:           okID,
						ComposedText: "Some AAPL news without meta.",
						MetaData:     d1,
						IsSuspicious: false,
					},
					{
						ID:           uuid.New(),
						ComposedText: "Some PLTR news.",
						MetaData:     d2,
						IsSuspicious: false,
					},
				},
			},
			want: []*archivist.News{
				{
					ID:           okID,
					ComposedText: "Some AAPL news without meta.",
					MetaData:     d1,
					IsSuspicious: false,
				},
			},
			wantErr: false,
		},
		{
			name: "Omit if all keys are empty",
			fields: fields{
				stocks: nil,
				options: &jobOptions{
					omitIfAllKeysEmpty: true,
				},
			},
			args: args{
				news: []*archivist.News{
					{
						ID:           uuid.New(),
						ComposedText: "Some AAPL news without meta.",
						MetaData:     emptyMeta,
						IsSuspicious: false,
					},
					{
						ID:           okID,
						ComposedText: "Some other AAPL news.",
						MetaData:     d1,
						IsSuspicious: false,
					},
				},
			},
			want: []*archivist.News{
				{
					ID:           okID,
					ComposedText: "Some other AAPL news.",
					MetaData:     d1,
					IsSuspicious: false,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid meta data",
			fields: fields{
				stocks:  nil,
				options: &jobOptions{},
			},
			args: args{
				news: []*archivist.News{
					{
						ID:           uuid.New(),
						ComposedText: "Some AAPL news without meta.",
						MetaData:     []byte("invalid"),
						IsSuspicious: false,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Omit filtered news",
			fields: fields{
				stocks:  nil,
				options: &jobOptions{},
			},
			args: args{
				news: []*archivist.News{
					{
						ID:           uuid.New(),
						ComposedText: "Some AAPL news about AAPL stock.",
						MetaData:     d1,
						IsFiltered:   true,
					},
					{
						ID:           okID,
						ComposedText: "Some other AAPL news.",
						MetaData:     d1,
						IsFiltered:   false,
					},
				},
			},
			want: []*archivist.News{
				{
					ID:           okID,
					ComposedText: "Some other AAPL news.",
					MetaData:     d1,
					IsFiltered:   false,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &Job{
				stocks:  tt.fields.stocks,
				options: tt.fields.options,
			}
			got, err := job.prepublishFilter(tt.args.news)
			if (err != nil) != tt.wantErr {
				t.Errorf("prepublishFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prepublishFilter() got = %v, want %v", got, tt.want)
			}
		})
	}
}
