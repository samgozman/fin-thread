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
				ID:           "726de2ac36a252f781db6af19c3c8039",
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
				ID:           "8ddb069f918346ccacecdf3fc1e1df9b",
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
			if len(tt.n.mapIDs()) != len(tt.want) {
				t.Errorf("mapIDs() = %v, want %v", tt.n.mapIDs(), tt.want)
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

func TestNews_contains(t *testing.T) {
	type args struct {
		keywords []string
	}
	tests := []struct {
		name   string
		fields News
		args   args
		want   bool
	}{
		{
			name: "contains one keyword",
			fields: News{
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
			fields: News{
				Title:       "Some news about United States",
				Description: "Read more about United States",
			},
			args: args{
				keywords: []string{"kek", "?"},
			},
			want: false,
		},
		{
			name: "contains none full words",
			fields: News{
				Title:       "Some news about United States",
				Description: "Read more about United States",
			},
			args: args{
				keywords: []string{"ted"},
			},
			want: false,
		},
		{
			name: "contains pronoun",
			fields: News{
				Title:       "'I'm not a cat': Lawyer struggles with Zoom kitten filter during court case",
				Description: "A lawyer in Texas has gone viral after accidentally appearing in court as a cat.",
			},
			args: args{
				keywords: []string{"i'm"},
			},
			want: true,
		},
		{
			name: "contains symbol",
			fields: News{
				Title:       "Some news about United States or not?",
				Description: "Read more about United States",
			},
			args: args{
				keywords: []string{"?"},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &News{
				Title:       tt.fields.Title,
				Description: tt.fields.Description,
			}
			if got := n.contains(tt.args.keywords); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewsList_ToContentJSON(t *testing.T) {
	tests := []struct {
		name    string
		n       NewsList
		want    string
		wantErr bool
	}{
		{
			name: "valid news list",
			n: NewsList{
				{
					ID:          "id1",
					Title:       "Some news about United States",
					Description: "Read more about United States",
				},
				{
					ID:           "id2",
					Title:        "Some news about kek",
					Description:  "Read more about kek",
					IsSuspicious: true,
				},
			},
			want:    `[{"id":"id1","title":"Some news about United States","description":"Read more about United States"},{"id":"id2","title":"Some news about kek","description":"Read more about kek"}]`,
			wantErr: false,
		},
		{
			name:    "empty news list",
			n:       NewsList{},
			want:    `[]`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.n.ToContentJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToContentJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ToContentJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewsList_RemoveFlagged(t *testing.T) {
	tests := []struct {
		name string
		n    NewsList
		want NewsList
	}{
		{
			name: "remove flagged",
			n: NewsList{
				{
					ID:          "id1",
					Title:       "Some news about United States",
					Description: "Read more about United States",
					IsFiltered:  true,
				},
				{
					ID:           "id2",
					Title:        "Some news about kek",
					Description:  "Read more about kek",
					IsSuspicious: true,
				},
				{
					ID:          "id3",
					Title:       "Some news about something",
					Description: "Read more about something",
				},
			},
			want: NewsList{
				{
					ID:          "id3",
					Title:       "Some news about something",
					Description: "Read more about something",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.n.RemoveFlagged(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveFlagged() = %v, want %v", got, tt.want)
			}
		})
	}
}
