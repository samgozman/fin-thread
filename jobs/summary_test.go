package jobs

import (
	"github.com/samgozman/fin-thread/composer"
	"testing"
	"time"
)

func Test_formatSummary(t *testing.T) {
	type args struct {
		headlines []*composer.SummarisedHeadline
		from      time.Time
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case many headlines",
			args: args{
				headlines: []*composer.SummarisedHeadline{
					{
						ID:      "1",
						Summary: "Someone warns text",
						Link:    "https://t.me/fin_thread/1",
						Verb:    "warns",
					},
					{
						ID:      "2",
						Summary: "Someone else warns text",
						Verb:    "warns",
					},
					{
						ID:      "3",
						Summary: "Someone else bonks text",
						Link:    "https://t.me/fin_thread/3",
					},
				},
				from: time.Now().Add(-7 * time.Hour),
			},
			want: "📓 #summary\n" +
				"What happened in the last 7 hours:\n" +
				"- Someone [warns](https://t.me/fin_thread/1) text\n" +
				"- Someone else warns text\n" +
				"- Someone else bonks text\n",
		},
		{
			name: "case no headlines",
			args: args{
				headlines: []*composer.SummarisedHeadline{},
				from:      time.Now(),
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatSummary(tt.args.headlines, tt.args.from); got != tt.want {
				t.Errorf("formatSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}
