package helper_test

import (
	"testing"

	"github.com/2Geigh/Herb/crawler/internal/helper"
)

func TestUrl_GetSecondAndTopLevelDomain(t *testing.T) {
	tests := []struct {
		name string
		url  helper.Url
		want helper.SecondAndTopLevelDomain
	}{
		{
			name: "HTTPS YouTube URL",
			url:  helper.Url("https://www.youtube.com/watch?v=dQw4w9WgXcQ"),
			want: helper.SecondAndTopLevelDomain("youtube.com"),
		},
		{
			name: "HTTP example URL",
			url:  helper.Url("http://www.example.com"),
			want: helper.SecondAndTopLevelDomain("example.com"),
		},
		{
			name: "URL with a path",
			url:  helper.Url("https://example.org/products/item"),
			want: helper.SecondAndTopLevelDomain("example.org"),
		},
		{
			name: "URL with a query string",
			url:  helper.Url("https://google.com/search?q=golang"),
			want: helper.SecondAndTopLevelDomain("google.com"),
		},
		{
			name: "URL with a fragment",
			url:  helper.Url("https://github.com/golang/go#readme"),
			want: helper.SecondAndTopLevelDomain("github.com"),
		},
		{
			name: "URL with multiple subdomains",
			url:  helper.Url("https://api.v1.example.com/users"),
			want: helper.SecondAndTopLevelDomain("example.com"),
		},
		{
			name: "WWW subdomain",
			url:  helper.Url("https://www.wikipedia.org/wiki/Go"),
			want: helper.SecondAndTopLevelDomain("wikipedia.org"),
		},
		{
			name: "Net top-level domain",
			url:  helper.Url("https://www.example.net"),
			want: helper.SecondAndTopLevelDomain("example.net"),
		},
		{
			name: "Educational domain",
			url:  helper.Url("https://www.example.edu/courses"),
			want: helper.SecondAndTopLevelDomain("example.edu"),
		},
		{
			name: "Government domain",
			url:  helper.Url("https://www.example.gov/services"),
			want: helper.SecondAndTopLevelDomain("example.gov"),
		},
		{
			name: "Country-code domain",
			url:  helper.Url("https://www.example.co.uk"),
			want: helper.SecondAndTopLevelDomain("co.uk"),
		},
		{
			name: "Country-code domain with subdomain",
			url:  helper.Url("https://shop.example.co.uk/products"),
			want: helper.SecondAndTopLevelDomain("co.uk"),
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
