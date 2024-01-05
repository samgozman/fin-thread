package jobs

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/samgozman/fin-thread/archivist/models"
	"github.com/samgozman/fin-thread/composer"
	"testing"
)

func Test_formatNewsWithComposedMeta(t *testing.T) {
	type args struct {
		n models.News
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
				n: models.News{
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
				n: models.News{
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
				n: models.News{
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
