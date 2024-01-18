package journalist

import (
	"reflect"
	"testing"
	"time"
)

func TestNewNews(t *testing.T) {
	type args struct {
		title        string
		description  string
		link         string
		date         string
		providerName string
	}
	tests := []struct {
		name    string
		args    args
		want    *News
		wantErr bool
	}{
		{
			name: "valid news",
			args: args{
				title:        "title",
				description:  "description",
				link:         "link",
				date:         "Mon, 02 Jan 2006 15:04:05 MST",
				providerName: "provider",
			},
			want: &News{
				ID:           "cbd261a703d9f7f5bf08f8ede0a1e99b",
				Title:        "title",
				Description:  "description",
				Link:         "link",
				Date:         time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
				ProviderName: "provider",
			},
			wantErr: false,
		},
		{
			name: "valid news with html tags",
			args: args{
				title:        "title <i>bonk</i>",
				description:  "description <b>bold</b> <i>S\\u0026P 500</i> <a href=\"link\">G&#38;T</a>",
				link:         "link",
				date:         "Mon, 02 Jan 2006 15:04:05 MST",
				providerName: "provider",
			},
			want: &News{
				ID:           "91e9909e2e1a1555d1d0aaca96aede63",
				Title:        "title bonk",
				Description:  "description bold S&P 500 G&T",
				Link:         "link",
				Date:         time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
				ProviderName: "provider",
			},
			wantErr: false,
		},
		{
			name: "invalid date",
			args: args{
				title:       "title",
				description: "description",
				link:        "link",
				date:        "invalid date",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newNews(tt.args.title, tt.args.description, tt.args.link, tt.args.date, tt.args.providerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("newNews() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newNews() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO: Add tests for ToJSON and ToContentJSON

func TestNewsList_FilterByKeywords(t *testing.T) {
	type args struct {
		keywords []string
	}
	tests := []struct {
		name string
		n    NewsList
		args args
		want NewsList
	}{
		{
			name: "filter by one keyword",
			n: NewsList{
				{
					ID:          "id1",
					Title:       "Some news about Uganda",
					Description: "Read more about Uganda",
				},
				{
					ID:          "id2",
					Title:       "Some news about United States",
					Description: "Read more about United States",
				},
			},
			args: args{
				keywords: []string{"United States"},
			},
			want: NewsList{
				{
					ID:          "id2",
					Title:       "Some news about United States",
					Description: "Read more about United States",
				},
			},
		},
		{
			name: "filter by multiple keywords",
			n: NewsList{
				{
					ID:          "id1",
					Title:       "Some news about Uganda",
					Description: "Read more about Uganda",
				},
				{
					ID:          "id2",
					Title:       "Some news about United States",
					Description: "Read more about United States",
				},
				{
					ID:          "id3",
					Title:       "Some news about United Kingdom",
					Description: "Read more about United Kingdom",
				},
			},
			args: args{
				keywords: []string{"United States", "United Kingdom"},
			},
			want: NewsList{
				{
					ID:          "id2",
					Title:       "Some news about United States",
					Description: "Read more about United States",
				},
				{
					ID:          "id3",
					Title:       "Some news about United Kingdom",
					Description: "Read more about United Kingdom",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.n.filterByKeywords(tt.args.keywords); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterByKeywords() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewsList_MapIDs(t *testing.T) {
	tests := []struct {
		name string
		n    NewsList
		want NewsList
	}{
		{
			name: "filter duplicates",
			n: NewsList{
				{
					ID:          "id1",
					Title:       "Some news",
					Description: "Read more",
				},
				{
					ID:          "id2",
					Title:       "Some news about United States",
					Description: "Read more about United States",
				},
				{
					ID:          "id1",
					Title:       "Some news",
					Description: "Read more",
				},
			},
			want: NewsList{
				{
					ID:          "id1",
					Title:       "Some news",
					Description: "Read more",
				},
				{
					ID:          "id2",
					Title:       "Some news about United States",
					Description: "Read more about United States",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.n.mapIDs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mapIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewsList_FlagByKeywords(t *testing.T) {
	type args struct {
		keywords []string
	}
	tests := []struct {
		name           string
		n              NewsList
		args           args
		wantFlaggedLen int
	}{
		{
			name: "flag by one keyword",
			n: NewsList{
				{
					ID:          "id1",
					Title:       "Some news about United States",
					Description: "Read more about United States",
				},
				{
					ID:          "id2",
					Title:       "Some news about kek",
					Description: "Read more about kek",
				},
				{
					ID:          "id3",
					Title:       "Some news about keking",
					Description: "Read more about keking",
				},
			},
			args: args{
				keywords: []string{"kek"},
			},
			wantFlaggedLen: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.n.flagByKeywords(tt.args.keywords)
			flaggedLen := 0
			for _, n := range tt.n {
				if n.IsSuspicious {
					flaggedLen++
				}
			}
			if flaggedLen != tt.wantFlaggedLen {
				t.Errorf("flagByKeywords() = %v, want %v", flaggedLen, tt.wantFlaggedLen)
			}
		})
	}
}

func TestNews_Contains(t *testing.T) {
	type fields struct {
		ID           string
		Title        string
		Description  string
		Link         string
		Date         time.Time
		ProviderName string
		IsSuspicious bool
	}
	type args struct {
		keywords []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "contains one keyword",
			fields: fields{
				Title:       "Some news about United States",
				Description: "Read more about United States",
			},
			args: args{
				keywords: []string{"united States"},
			},
			want: true,
		},
		{
			name: "contains none",
			fields: fields{
				Title:       "Some news about United States",
				Description: "Read more about United States",
			},
			args: args{
				keywords: []string{"kek"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &News{
				Title:       tt.fields.Title,
				Description: tt.fields.Description,
			}
			if got := n.Contains(tt.args.keywords); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
