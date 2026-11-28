package api

// mustNewClient is NewClient for tests: it panics when the options are invalid.
func mustNewClient(opts ClientOptions) *Client {
	client, err := NewClient(opts)
	if err != nil {
		panic(err)
	}
	return client
}
