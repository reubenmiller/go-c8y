package model

import "time"

// LoginOption
type LoginOption struct {
	// SSO specific. Describes the fields in the access token from the external server containing user information
	AccessTokenToUserDataMapping *struct {
		// The name of the field containing the user's email
		EmailClaimName string `json:"emailClaimName,omitempty"`

		// The name of the field containing the user's first name
		FirstNameClaimName string `json:"firstNameClaimName,omitempty"`

		// The name of the field containing the user's last name
		LastNameClaimName string `json:"lastNameClaimName,omitempty"`

		// The name of the field containing the user's phone number
		PhoneNumberClaimName string `json:"phoneNumberClaimName,omitempty"`
	} `json:"accessTokenToUserDataMapping,omitempty"`

	// SSO specific. Token audience
	Audience string `json:"audience,omitempty"`

	// SSO specific. Request to the external authorization server used by the Cumulocity platform to obtain an authorization code
	AuthorizationRequest map[string]any `json:"authorizationRequest,omitempty"`

	// For basic authentication case only
	AuthenticationRestrictions *struct {
		// List of types of clients which are not allowed to use basic authentication. Currently the only supported option is WEB_BROWSERS
		ForbiddenClients []string `json:"forbiddenClients,omitempty"`

		// List of user agents, passed in User-Agent HTTP header, which are blocked if basic authentication is used
		ForbiddenUserAgents []string `json:"forbiddenUserAgents,omitempty"`

		// List of user agents, passed in User-Agent HTTP header, which are allowed to use basic authentication
		TrustedUserAgents []string `json:"trustedUserAgents,omitempty"`
	} `json:"authenticationRestrictions,omitempty"`

	// SSO specific. Information for the UI about the name displayed on the external server login button
	ButtonName string `json:"buttonName,omitempty"`

	// SSO specific. The identifier of the Cumulocity tenant on the external authorization server
	ClientID string `json:"clientId,omitempty"`

	// The authentication configuration grant type identifier
	// Enum: "AUTHORIZATION_CODE" "PASSWORD"
	GrantType string `json:"grantType,omitempty"`

	// SSO specific. External token issuer
	Issuer string `json:"issuer,omitempty"`

	// SSO specific. Request to the external authorization server used by the Cumulocity platform to logout the user
	LogoutRequest map[string]any `json:"logoutRequest,omitempty"`

	// Indicates whether the configuration is only accessible to the management tenant
	OnlyManagementTenantAccess bool `json:"onlyManagementTenantAccess,omitempty"`

	// SSO specific. Describes the process of internal user creation during login with the external authorization server
	OnNewUser map[string]any `json:"onNewUser,omitempty"`

	// The name of the authentication provider
	ProviderName string `json:"providerName,omitempty"`

	// SSO specific. URL used for redirecting to the Cumulocity platform. Do not set or leave it empty to allow SSO flow to be controlled by client (UI) applications
	RedirectToPlatform string `json:"redirectToPlatform,omitempty"`

	// SSO specific. Request to the external authorization server used by the Cumulocity platform to obtain a refresh token
	RefreshRequest map[string]any `json:"refreshRequest,omitempty"`

	// The session configuration properties are only available for OAI-Secure. See Platform administration > Authentication > Basic settings > OAI Secure session configuration in the Cumulocity user documentation
	SessionConfiguration *struct {
		// Maximum session duration (in milliseconds) during which a user does not have to login again
		AbsoluteTimeoutMillis int64 `json:"absoluteTimeoutMillis,omitempty"`

		// Maximum number of parallel sessions for one user
		MaximumNumberOfParallelSessions int64 `json:"maximumNumberOfParallelSessions,omitempty"`

		// Amount of time before a token expires (in milliseconds) during which the token may be renewed
		RenewalTimeoutMillis int64 `json:"renewalTimeoutMillis,omitempty"`

		// Switch to turn additional user agent verification on or off during the session
		UserAgentValidationRequired bool `json:"userAgentValidationRequired,omitempty"`
	} `json:"sessionConfiguration,omitempty"`

	// SSO specific and authorization server dependent. Describes the method of access token signature verification on the Cumulocity platform.
	SignatureVerificationConfig *struct {
		// AAD signature verification configuration
		AAD *struct {
			// URL used to retrieve the public key used for signature verification
			PublicKeyDiscoveryUrl string `json:"publicKeyDiscoveryUrl,omitempty"`
		} `json:"aad,omitempty"`

		// ADFS manifest signature verification configuration
		ADFSManifest *struct {
			// The URI to the manifest resource
			ManifestURL string `json:"manifestUrl,omitempty"`
		} `json:"adfsManifest,omitempty"`

		// The address of the endpoint which is used to retrieve the public key used to verify the JWT access token signature
		JWKS *struct {
			// The URI to the public key resource
			JwksURL string `json:"jwksUrl,omitempty"`
		} `json:"jwks,omitempty"`

		// Describes the process of verification of JWT access token with the public keys embedded in the provided X.509 certificates
		Manual *struct {
			// The name of the field in the JWT access token containing the certificate identifier
			CertIDField string `json:"certIdField,omitempty"`

			// Indicates whether the certificate identifier should be read from the JWT access token
			CertIDFromField bool `json:"certIdFromField,omitempty"`

			// Details of the certificates
			Certificates *struct {
				// The signing algorithm of the JWT access token
				// Enum: "RSA" "PCKS"
				SigningAlgorithm string `json:"alg,omitempty"`

				// The public key certificate
				PublicKey string `json:"publicKey,omitempty"`

				// The validity start date of the certificate
				ValidFrom time.Time `json:"validFrom,omitempty,omitzero"`

				// The expiry date of the certificate
				ValidTo time.Time `json:"validTill,omitempty,omitzero"`
			} `json:"certificates,omitempty"`
		} `json:"manual,omitempty"`
	} `json:"signatureVerificationConfig,omitempty"`

	// SSO specific. Template name used by the UI
	Template string `json:"template,omitempty"`

	// SSO specific. Request to the external authorization server used by the Cumulocity platform to obtain an access token
	TokenRequest *struct {
		// Body of the request
		Body string `json:"body,omitempty"`

		// It is possible to add an arbitrary number of headers as a list of key-value string pairs,
		// for example, "header": "value"
		Headers map[string]string `json:"headers,omitempty"`

		// HTTP request method
		Method string `json:"method,omitempty"`

		// Requested operation
		Operation string `json:"operation,omitempty"`

		// Parameters of the request
		RequestParams map[string]string `json:"requestParams,omitempty"`

		// Target of the request described as a URL
		URL string `json:"url,omitempty"`
	} `json:"tokenRequest,omitempty"`

	// The authentication configuration type. Note that the value is case insensitive
	// Enum: "BASIC" "OAUTH2" "OAUTH2_INTERNAL"
	Type string `json:"type,omitempty"`

	// Unique identifier of this login option
	ID string `json:"id,omitempty"`

	// If set to true, user data and the userId are retrieved using the claims from the id_token; otherwise, they are based on the access_token
	UseIDToken bool `json:"useIdToken,omitempty"`

	// SSO specific. Points to the field in the obtained JWT access token that should be used as the username in the Cumulocity platform
	UserIDConfig *struct {
		// The name of the field containing the JWT
		JwtField string `json:"jwtField,omitempty"`

		// Used only when useConstantValue is set to true
		// ConstantValue string `json:"constantValue,omitempty"`

		// Not recommended. If set to true, all SSO users will share one account in the Cumulocity platform
		// UseConstantValue string `json:"useConstantValue,omitempty"`
	} `json:"userIdConfig,omitempty"`

	// Indicates whether user data are managed internally by the Cumulocity platform or by an external server. Note that the value is case insensitive
	// Enum: "INTERNAL" "REMOTE"
	UserManagementSource string `json:"userManagementSource,omitempty"`

	// Information for the UI if the respective authentication form should be visible for the user
	VisibleOnLoginPage bool `json:"visibleOnLoginPage,omitempty"`

	ExternalTokenConfig *struct {
		// Indicates whether authentication is enabled or disabled
		Enabled bool `json:"enabled,omitempty"`

		// Points to the claim of the access token from the authorization server that must be used as the username in the Cumulocity platform
		UserIDConfig *struct {
			// The name of the field containing the JWT
			JwtField string `json:"jwtField,omitempty"`

			// Used only when useConstantValue is set to true
			// ConstantValue string `json:"constantValue,omitempty"`

			// Not recommended. If set to true, all SSO users will share one account in the Cumulocity platform
			// UseConstantValue string `json:"useConstantValue,omitempty"`
		} `json:"userIdConfig,omitempty"`

		// If set to true, the access token is validated against the authorization server by way of introspection or user info request.
		ValidationRequired bool `json:"validationRequired,omitempty"`

		// The method of validation of the access token
		// Enum: "INTROSPECTION" "USERINFO"
		ValidationMethod string `json:"validationMethod,omitempty"`

		// It is possible to add an arbitrary number of parameters as a list of key-value string pairs, for example, "parameter": "value"
		TokenValidationRequest map[string]string `json:"tokenValidationRequest,omitempty"`

		// The frequency (in Minutes) in which Cumulocity sends a validation request to authorization server. The recommended frequency is 1 minute
		AccessTokenValidityCheckIntervalInMinutes int64 `json:"accessTokenValidityCheckIntervalInMinutes,omitempty"`

		// Not recommended. If set to true, all SSO users will share one account in the Cumulocity platform
		// UseConstantValue string `json:"useConstantValue,omitempty"`
	} `json:"externalTokenConfig,omitempty"`
}

// EventCollection collection of events
type LoginOptionCollection struct {
	*BaseResponse

	LoginOptions []LoginOption `json:"loginOptions"`
}
