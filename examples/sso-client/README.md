# SSO Client Example

This example demonstrates how to use OAuth2 Authorization Code Flow to authenticate with Cumulocity IoT and retrieve data.

## Prerequisites

1. A Cumulocity IoT tenant with OAuth2 configured
2. An OAuth2 client application registered in your tenant
3. The client application should have the following redirect URI configured (e.g. `http://127.0.0.1:5001/callback`)


## How it works

1. **Authorization URL Generation**: The application generates an OAuth2 authorization URL
2. **User Authentication**: You visit the URL in your browser and authenticate
3. **Authorization Code**: After authentication, you're redirected to `http://127.0.0.1:5001/callback` with an authorization code
4. **Local Server**: The application runs a local HTTP server to capture the authorization code
5. **Token Exchange**: The code is exchanged for an access token
6. **API Access**: The token is used to authenticate API requests

## Running the Example

```sh
export C8Y_HOST="https://example.cumulocity.com"
go run main.go
```

Or you can customize the local callback url using the following environment variables:

```sh
export C8Y_HOST="https://example.cumulocity.com"
export C8Y_CALLBACK_URL="http://127.0.0.1:5001/callback"
go run main.go
```

## Troubleshooting

- **Port 5001 already in use**: Change the redirect URI by setting the `C8Y_CALLBACK_URL` environment variable
