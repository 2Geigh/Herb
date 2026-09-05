package models_test

import (
	"testing"

	"github.com/2Geigh/Herb/crawler/internal/models"
)

func TestUrl_GetSecondAndTopLevelDomain(t *testing.T) {
	tests := []struct {
		name string
		url  models.Url
		want models.SecondAndTopLevelDomain
	}{
		{
			name: "HTTPS YouTube URL",
			url:  models.Url("https://www.youtube.com/watch?v=dQw4w9WgXcQ"),
			want: models.SecondAndTopLevelDomain("youtube.com"),
		},
		{
			name: "HTTP example URL",
			url:  models.Url("http://www.example.com"),
			want: models.SecondAndTopLevelDomain("example.com"),
		},
		{
			name: "URL with a path",
			url:  models.Url("https://example.org/products/item"),
			want: models.SecondAndTopLevelDomain("example.org"),
		},
		{
			name: "URL with a query string",
			url:  models.Url("https://google.com/search?q=golang"),
			want: models.SecondAndTopLevelDomain("google.com"),
		},
		{
			name: "URL with a fragment",
			url:  models.Url("https://github.com/golang/go#readme"),
			want: models.SecondAndTopLevelDomain("github.com"),
		},
		{
			name: "URL with multiple subdomains",
			url:  models.Url("https://api.v1.example.com/users"),
			want: models.SecondAndTopLevelDomain("example.com"),
		},
		{
			name: "WWW subdomain",
			url:  models.Url("https://www.wikipedia.org/wiki/Go"),
			want: models.SecondAndTopLevelDomain("wikipedia.org"),
		},
		{
			name: "Net top-level domain",
			url:  models.Url("https://www.example.net"),
			want: models.SecondAndTopLevelDomain("example.net"),
		},
		{
			name: "Educational domain",
			url:  models.Url("https://www.example.edu/courses"),
			want: models.SecondAndTopLevelDomain("example.edu"),
		},
		{
			name: "Government domain",
			url:  models.Url("https://www.example.gov/services"),
			want: models.SecondAndTopLevelDomain("example.gov"),
		},
		{
			name: "Country-code domain",
			url:  models.Url("https://www.example.co.uk"),
			want: models.SecondAndTopLevelDomain("co.uk"),
		},
		{
			name: "Country-code domain with subdomain",
			url:  models.Url("https://shop.example.co.uk/products"),
			want: models.SecondAndTopLevelDomain("co.uk"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.url.GetSecondAndTopLevelDomain()

			if got != tt.want {
				t.Errorf(
					"GetSecondAndTopLevelDomain() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}
