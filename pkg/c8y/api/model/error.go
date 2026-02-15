package model

import (
	"fmt"
	"net/url"
)

/*
An ErrorResponse reports one or more errors caused by an API request.
*/
type ErrorResponse struct {
	RequestMethod      string   `json:"-"`
	RequestURL         *url.URL `json:"-"`
	ResponseStatusCode int64    `json:"-"`

	ErrorType string `json:"error,omitempty"`   // Error type formatted as "<<resource type>>/<<error name>>"". For example, an object not found in the inventory is reported as "inventory/notFound".
	Message   string `json:"message,omitempty"` // error message
	Info      string `json:"info,omitempty"`    // URL to an error description on the Internet.

	// Error details. Only available in DEBUG mode.
	Details *struct {
		ExceptionClass      string `json:"exceptionClass,omitempty"`
		ExceptionMessage    string `json:"exceptionMessage,omitempty"`
		ExceptionStackTrace string `json:"exceptionStackTrace,omitempty"`
	} `json:"details,omitempty"`
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v %v",
		r.RequestMethod, sanitizeURL(r.RequestURL),
		r.ResponseStatusCode, r.ErrorType, r.Message)
}

// sanitizeURL redacts the client_secret parameter from the URL which may be
// exposed to the user.
func sanitizeURL(uri *url.URL) *url.URL {
	if uri == nil {
		return nil
	}
	params := uri.Query()
	if len(params.Get("client_secret")) > 0 {
		params.Set("client_secret", "REDACTED")
		uri.RawQuery = params.Encode()
	}
	return uri
}
