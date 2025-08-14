package wsurl

import (
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
)

func Test_GetEndpoint(t *testing.T) {
	cases := []struct {
		Host           string
		Path           string
		ExpectedScheme string
		ExpectedHost   string
		ExpectedPath   string
	}{
		{
			Host:           "http://cumulocity:8111",
			Path:           "foo/bar",
			ExpectedScheme: "ws",
			ExpectedHost:   "cumulocity:8111",
			ExpectedPath:   "/foo/bar",
		},
		{
			Host:           "https://cumulocity:8111",
			Path:           "foo/bar",
			ExpectedScheme: "wss",
			ExpectedHost:   "cumulocity:8111",
			ExpectedPath:   "/foo/bar",
		},
		{
			Host:           "https://cumulocity:8111/",
			Path:           "foo/bar",
			ExpectedScheme: "wss",
			ExpectedHost:   "cumulocity:8111",
			ExpectedPath:   "/foo/bar",
		},
		{
			Host:           "https://cumulocity:8111/some/nested",
			Path:           "foo/bar",
			ExpectedScheme: "wss",
			ExpectedHost:   "cumulocity:8111",
			ExpectedPath:   "/some/nested/foo/bar",
		},
	}
	for _, item := range cases {
		endpoint, err := GetWebsocketURL(item.Host, item.Path)
		testingutils.Ok(t, err)
		testingutils.Equals(t, item.ExpectedScheme, endpoint.Scheme)
		testingutils.Equals(t, item.ExpectedHost, endpoint.Host)
		testingutils.Equals(t, item.ExpectedPath, endpoint.Path)
	}
}
