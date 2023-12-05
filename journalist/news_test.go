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
				description:  "description <b>bold</b> <i>italic</i> <a href=\"link\">link</a>",
				link:         "link",
				date:         "Mon, 02 Jan 2006 15:04:05 MST",
				providerName: "provider",
			},
			want: &News{
				ID:           "309e1c0cfc773eccc628ba376378eaa1",
				Title:        "title bonk",
				Description:  "description bold italic link",
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
			got, err := NewNews(tt.args.title, tt.args.description, tt.args.link, tt.args.date, tt.args.providerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNews() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNews() = %v, want %v", got, tt.want)
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
			if got := tt.n.FilterByKeywords(tt.args.keywords); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterByKeywords() = %v, want %v", got, tt.want)
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
			if got := tt.n.MapIDs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapIDs() = %v, want %v", got, tt.want)
			}
		})
	}
}
