// Package main demonstrates how to use LoginOptions.Method and
// LoginOptions.Preference to let CLI users control the Cumulocity login flow.
//
// USAGE
//
//	cli-login-preference [flags]
//
// FLAGS
//
//	-method  string   Enforce exactly one login method (strict).
//	                  Fails immediately when the required credentials are
//	                  absent, or when the tenant has no external OAuth2
//	                  provider for SSO-based methods.
//	                  Values: BASIC | OAUTH2_INTERNAL | CERTIFICATE |
//	                          OAUTH2_DEVICE_FLOW | OAUTH2_BROWSER_FLOW
//
//	-prefer  string   Add a method to the ordered preference list (repeatable).
//	                  Methods are tried left-to-right; the first one that
//	                  succeeds is used. Methods whose local prerequisites
//	                  are absent (no credentials, no client cert, or tenant
//	                  has no external OAuth2 provider) are silently skipped.
//	                  Example: -prefer OAUTH2_INTERNAL -prefer OAUTH2_DEVICE_FLOW
//
//	-browser-addr     Callback URL for the OAUTH2_BROWSER_FLOW redirect.
//	                  Accepted forms:
//	                    http://127.0.0.1:5001/callback  (explicit)
//	                    127.0.0.1:5001/login            (scheme inferred)
//	                    127.0.0.1:5001                  (path defaults to /callback)
//	                  Default: "127.0.0.1:5001" --> http://127.0.0.1:5001/callback.
//	                  Must match the redirect URI registered in the SSO provider.
//
//	-browser-pkce     Enable PKCE (Proof Key for Code Exchange, RFC 7636) for the
//	                  OAUTH2_BROWSER_FLOW. Appends code_challenge / code_challenge_method=S256
//	                  to the IdP authorization URL and sends code_verifier in the
//	                  token exchange. Requires support from both the external IdP
//	                  and Cumulocity's token endpoint (may not be supported yet).
//
//	-host            Cumulocity base URL, e.g. https://mytenant.cumulocity.com.
//	                  Overrides $C8Y_BASEURL / $C8Y_URL / $C8Y_HOST.
//
//	-user    string   Username. Overrides $C8Y_USER / $C8Y_USERNAME.
//	                  Prompted interactively when required and not provided.
//
//	-password string  Password. Overrides $C8Y_PASSWORD.
//	                  Prompted interactively when required and not provided.
//
//	-tenant  string   Tenant ID (optional). Overrides $C8Y_TENANT.
//
//	-no-env           Do not read any values from environment variables.
//	                  All connection details must be supplied via flags.
//
//	-debug            Enable debug mode with full HTTP request/response logging.
//	                  Sensitive headers (e.g. Authorization) are shown unredacted.
//
//	-totp-secret      Base32 TOTP secret for unattended code generation during
//	                  OAUTH2_INTERNAL login. Falls back to $C8Y_TOTP_SECRET
//	                  (unless -no-env is set).
//	                  WARNING: storing a TOTP secret alongside credentials
//	                  removes the second-factor security benefit.
//
// ENVIRONMENT VARIABLES
//
//	C8Y_BASEURL    https://mytenant.cumulocity.com
//	C8Y_USER       Cumulocity username
//	C8Y_PASSWORD   Cumulocity password
//	C8Y_TENANT     Tenant ID (optional, e.g. t12345)
//	C8Y_TOKEN      Bearer token (skips username/password login)
//
// # EXAMPLES
//
// Force Basic auth (special-case integrations only):
//
//	C8Y_USER=admin C8Y_PASSWORD=secret \
//	  cli-login-preference -method BASIC
//
// Force OAI-Secure (username+password -> short-lived token):
//
//	cli-login-preference -method OAUTH2_INTERNAL
//
// Open a browser for the SSO authorization code flow:
//
//	cli-login-preference -method OAUTH2_BROWSER_FLOW -browser-addr localhost:8080
//
// Prefer internal SSO; fall back to device flow when credentials are absent:
//
//	cli-login-preference \
//	  -prefer OAUTH2_INTERNAL \
//	  -prefer OAUTH2_DEVICE_FLOW
//
// Device flow first; Basic auth as last resort:
//
//	C8Y_USER=admin C8Y_PASSWORD=secret \
//	  cli-login-preference \
//	    -prefer OAUTH2_DEVICE_FLOW \
//	    -prefer BASIC
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/mdp/qrterminal/v3"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/currenttenant"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users/currentuser/totp"
	"github.com/reubenmiller/go-c8y/pkg/oauth/device"
	"golang.org/x/term"
)

// ---------------------------------------------------------------------------
// stringSliceFlag is a flag.Value that lets -prefer be given multiple times.
// ---------------------------------------------------------------------------

type stringSliceFlag []string

func (s *stringSliceFlag) String() string { return strings.Join(*s, ", ") }
func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// -- Flags ----------------------------------------------------------------
	var (
		hostFlag        string
		userFlag        string
		passwordFlag    string
		tenantFlag      string
		noEnvFlag       bool
		debugFlag       bool
		methodFlag      string
		preferFlags     stringSliceFlag
		browserAddrFlag = "http://127.0.0.1:5001/callback"
		browserPKCEFlag bool
		totpSecretFlag  string
	)

	flag.StringVar(&hostFlag, "host", "",
		"Cumulocity base URL, e.g. https://mytenant.cumulocity.com.\n"+
			"Overrides $C8Y_BASEURL / $C8Y_URL / $C8Y_HOST.")
	flag.StringVar(&userFlag, "user", "",
		"Username. Overrides $C8Y_USER / $C8Y_USERNAME.\n"+
			"Prompted interactively when required and not provided.")
	flag.StringVar(&passwordFlag, "password", "",
		"Password. Overrides $C8Y_PASSWORD.\n"+
			"Prompted interactively when required and not provided.")
	flag.StringVar(&tenantFlag, "tenant", "",
		"Tenant ID (optional). Overrides $C8Y_TENANT.")
	flag.BoolVar(&noEnvFlag, "no-env", false,
		"Do not read any values from environment variables.\n"+
			"All connection details must be supplied via flags.")
	flag.BoolVar(&debugFlag, "debug", false,
		"Enable debug mode with full HTTP request/response logging.\n"+
			"Sensitive headers (e.g. Authorization) are shown unredacted.")
	flag.StringVar(&methodFlag, "method", "",
		"Enforce exactly one login method (strict).\n"+
			"Values: BASIC | OAUTH2_INTERNAL | CERTIFICATE | OAUTH2_DEVICE_FLOW | OAUTH2_BROWSER_FLOW")
	flag.Var(&preferFlags, "prefer",
		"Add a method to the ordered preference list (repeatable).\n"+
			"The first available method in the list is used.")
	flag.StringVar(&browserAddrFlag, "browser-addr", browserAddrFlag,
		"Callback URL for the OAUTH2_BROWSER_FLOW redirect, e.g.\n"+
			"  http://127.0.0.1:5001/callback  (explicit)\n"+
			"  127.0.0.1:5001/login            (scheme inferred as http)\n"+
			"  127.0.0.1:5001                  (path defaults to /callback)\n"+
			"Must match the redirect URI registered in your SSO provider.")
	flag.BoolVar(&browserPKCEFlag, "browser-pkce", false,
		"Enable PKCE (RFC 7636) for OAUTH2_BROWSER_FLOW.\n"+
			"Requires support from both the external IdP and Cumulocity's token endpoint.")
	flag.StringVar(&totpSecretFlag, "totp-secret", "",
		"Base32 TOTP secret for automatic code generation (automation only).\n"+
			"Falls back to $C8Y_TOTP_SECRET (unless -no-env is set).")

	flag.Usage = printUsage
	flag.Parse()

	// Apply env-var defaults for values not explicitly provided via flags.
	if !noEnvFlag && totpSecretFlag == "" {
		totpSecretFlag = os.Getenv("C8Y_TOTP_SECRET")
	}

	// -- Validate flags -------------------------------------------------------
	if methodFlag != "" && len(preferFlags) > 0 {
		fmt.Fprintln(os.Stderr, "error: -method and -prefer are mutually exclusive")
		os.Exit(1)
	}

	var strictMethod authentication.LoginMethod
	if methodFlag != "" {
		parsed, err := authentication.ParseLoginMethod(methodFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid -method %q: %s\n", methodFlag, err)
			fmt.Fprintln(os.Stderr, "valid methods: BASIC, OAUTH2_INTERNAL, CERTIFICATE, OAUTH2_DEVICE_FLOW, OAUTH2_BROWSER_FLOW")
			os.Exit(1)
		}
		strictMethod = parsed
	}

	var preferMethods []authentication.LoginMethod
	for _, p := range preferFlags {
		m, err := authentication.ParseLoginMethod(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid -prefer %q: %s\n", p, err)
			fmt.Fprintln(os.Stderr, "valid methods: BASIC, OAUTH2_INTERNAL, CERTIFICATE, OAUTH2_DEVICE_FLOW, OAUTH2_BROWSER_FLOW")
			os.Exit(1)
		}
		preferMethods = append(preferMethods, m)
	}

	ctx := context.Background()

	// -- Build client ---------------------------------------------------------
	baseURL := hostFlag
	var auth authentication.AuthOptions
	if !noEnvFlag {
		if baseURL == "" {
			baseURL = authentication.HostFromEnvironment()
		}
		auth = authentication.FromEnvironment()
	}
	// Explicit flags always win over environment values.
	if userFlag != "" {
		auth.Username = userFlag
	}
	if passwordFlag != "" {
		auth.Password = passwordFlag
	}
	if tenantFlag != "" {
		auth.Tenant = tenantFlag
	}

	// Prompt for the host now if it's still unknown; it is always required
	// regardless of login method.
	if baseURL == "" {
		fmt.Fprint(os.Stderr, "Host (e.g. https://mytenant.cumulocity.com): ")
		sc := bufio.NewScanner(os.Stdin)
		if sc.Scan() {
			baseURL = strings.TrimSpace(sc.Text())
		}
	}

	client := api.NewClient(api.ClientOptions{
		BaseURL:       baseURL,
		Auth:          auth,
		Debug:         debugFlag,
		ShowSensitive: debugFlag,
	})

	// -- Print what we are about to do ----------------------------------------
	switch {
	case strictMethod != "":
		fmt.Fprintf(os.Stderr, "Login method (strict): %s\n\n", strictMethod)
	case len(preferMethods) > 0:
		names := make([]string, len(preferMethods))
		for i, m := range preferMethods {
			names[i] = string(m)
		}
		fmt.Fprintf(os.Stderr, "Login preference: %s\n\n", strings.Join(names, " -> "))
	default:
		fmt.Fprintf(os.Stderr, "Login method: default (OAUTH2_INTERNAL with TFA support)\n\n")
	}

	// -- Build LoginOptions ---------------------------------------------------
	//
	// Method and Preference are mutually exclusive:
	//   Method     - strict: exactly this flow must succeed or login fails.
	//   Preference - soft:   tries each in order, skipping unavailable ones.
	//
	// BrowserFlow and DeviceFlow configure the SSO flows when they are selected.
	// The TFA callbacks (QRCode, TOTPCode, SMSCode, PasswordChange) are used
	// only when OAUTH2_INTERNAL is the active flow.
	var usedMethod authentication.LoginMethod
	loginOpts := api.LoginOptions{
		// --- Method selection ------------------------------------------------
		Method:     strictMethod,
		Preference: preferMethods,

		// OnSuccess records which method in the preference list actually won.
		OnSuccess: func(m authentication.LoginMethod) { usedMethod = m },

		// CredentialPrompt is called lazily by the SDK when a method that needs
		// credentials is about to run but they are missing. Different methods
		// require different fields:
		//   BASIC / OAUTH2_INTERNAL  →  Username, Password
		//   CERTIFICATE              →  Certificate, CertificateKey (file paths)
		CredentialPrompt: func(ctx context.Context, method authentication.LoginMethod, auth *authentication.AuthOptions) error {
			switch method {
			case authentication.LoginMethodBasic, authentication.LoginMethodOAuth2Internal:
				if auth.Username == "" || auth.Password == "" {
					fmt.Fprintf(os.Stderr, "Login Method: %s\n", method)
				}
				if auth.Username == "" {

					fmt.Fprint(os.Stderr, "Username: ")
					sc := bufio.NewScanner(os.Stdin)
					if sc.Scan() {
						auth.Username = strings.TrimSpace(sc.Text())
					}
				}
				if auth.Password == "" {
					fmt.Fprint(os.Stderr, "Password: ")
					var err error
					auth.Password, err = readSecret()
					if err != nil {
						return fmt.Errorf("reading password: %w", err)
					}
				}
			case authentication.LoginMethodCertificate:
				if auth.Certificate == "" {
					fmt.Fprint(os.Stderr, "Certificate file path: ")
					sc := bufio.NewScanner(os.Stdin)
					if sc.Scan() {
						auth.Certificate = strings.TrimSpace(sc.Text())
					}
				}
				if auth.CertificateKey == "" {
					fmt.Fprint(os.Stderr, "Certificate key file path: ")
					sc := bufio.NewScanner(os.Stdin)
					if sc.Scan() {
						auth.CertificateKey = strings.TrimSpace(sc.Text())
					}
				}
			}
			return nil
		},

		// --- SSO flow configuration ------------------------------------------

		// BrowserFlow is used when OAUTH2_BROWSER_FLOW is selected.
		// CallbackURL must exactly match the redirect URI registered in the
		// SSO provider. Accepted forms:
		//   http://127.0.0.1:5001/callback  (explicit)
		//   127.0.0.1:5001/login            (scheme inferred)
		//   127.0.0.1:5001                  (path defaults to /callback)
		BrowserFlow: &api.BrowserFlowOptions{
			CallbackURL: browserAddrFlag,
			PKCE:        browserPKCEFlag,
		},

		// DeviceFlow is used when OAUTH2_DEVICE_FLOW is selected.
		// Zero-value causes endpoints to be auto-discovered via OpenID Connect.
		DeviceFlow: &api.DeviceFlowOptions{
			DisplayFunc: func(code *device.CodeResponse) error {
				// Use the complete URI (includes the user code) when available —
				// a single scan is all the user needs.
				urlToEncode := code.VerificationURIComplete
				if urlToEncode == "" {
					urlToEncode = code.VerificationURI
				}
				fmt.Fprintf(os.Stderr, "\nScan the QR code to authorise this device:\n\n")
				qrterminal.GenerateWithConfig(urlToEncode, qrterminal.Config{
					Level:      qrterminal.M,
					Writer:     os.Stderr,
					HalfBlocks: true,
					QuietZone:  1,
				})
				fmt.Fprintf(os.Stderr, "\n  Code: %s\n", code.UserCode)
				fmt.Fprintf(os.Stderr, "  URL:  %s\n\n", urlToEncode)
				fmt.Fprint(os.Stderr, "Open in browser? [Y/n]: ")
				sc := bufio.NewScanner(os.Stdin)
				if sc.Scan() {
					if answer := strings.ToLower(strings.TrimSpace(sc.Text())); answer == "" || answer == "y" {
						if err := api.DefaultBrowserOpen(urlToEncode); err != nil {
							fmt.Fprintf(os.Stderr, "  (could not open browser: %v)\n", err)
						}
					}
				}
				fmt.Fprintln(os.Stderr)
				return nil
			},
		},

		// --- OAUTH2_INTERNAL TFA callbacks -----------------------------------

		// TOTPSecret enables fully unattended code generation.
		// WARNING: only use in machine/automation scenarios; storing a TOTP
		// secret alongside credentials removes the second-factor security benefit.
		TOTPSecret: totpSecretFlag,

		// QRCode is called once during first-time TOTP enrollment.
		QRCode: func(otpauthURL, secret string) {
			fmt.Fprintf(os.Stderr, "\nScan the QR code below with your authenticator app:\n\n")
			qrterminal.GenerateWithConfig(otpauthURL, qrterminal.Config{
				Level:      qrterminal.M,
				Writer:     os.Stderr,
				HalfBlocks: true,
				QuietZone:  1,
			})
			fmt.Fprintf(os.Stderr, "\n  Secret: %s\n", secret)
			fmt.Fprintf(os.Stderr, "  URL:    %s\n\n", otpauthURL)
		},

		// TOTPCode is called interactively when a TOTP code is needed.
		TOTPCode: promptTOTPCode,

		// SMSCode is called when the server sends an SMS PIN challenge.
		SMSCode: promptSMSCode,

		// PasswordChange is called when the server forces a password reset.
		PasswordChange: promptNewPassword,
	}

	// -- Login ----------------------------------------------------------------
	_, err := client.LoginWithOptions(ctx, loginOpts)
	if err != nil {
		slog.Error("Login failed", "err", err)
		os.Exit(1)
	}

	// -- Verify with a lightweight API call -----------------------------------
	tenantResult := client.Tenants.Current.Get(ctx, currenttenant.GetOptions{})
	if tenantResult.Err != nil {
		slog.Error("API call failed", "err", tenantResult.Err)
		os.Exit(1)
	}

	userResult := client.Users.CurrentUser.Get(ctx)
	if userResult.Err != nil {
		slog.Error("API call failed", "err", userResult.Err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Authenticated")
	fmt.Fprintf(os.Stderr, "  Method:  %s\n", usedMethod)
	fmt.Fprintf(os.Stderr, "  User:    %s\n", userResult.Data.UserName())
	fmt.Fprintf(os.Stderr, "  Tenant:  %s\n", tenantResult.Data.Name())
	fmt.Fprintf(os.Stderr, "  Domain:  %s\n", tenantResult.Data.DomainName())

	// -- Smoke-test: device count ---------------------------------------------
	devResult := client.Devices.List(ctx, devices.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			WithTotalElements: true,
			PageSize:          1,
		},
	})
	if devResult.Err != nil {
		slog.Error("Devices API call failed", "err", devResult.Err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "  Devices: %d\n", devResult.TotalElements())
}

// ---------------------------------------------------------------------------
// Interactive prompt helpers
// ---------------------------------------------------------------------------

// promptTOTPCode is a totp.TOTPCodeFunc that reads a 6-digit code from the
// terminal (hidden input on a real TTY; plain line read when piped).
func promptTOTPCode(_ context.Context, challenge totp.TOTPChallenge) (string, error) {
	if challenge.IsSetup {
		fmt.Fprint(os.Stderr, "Enter the 6-digit code from your authenticator app to verify enrollment: ")
	} else {
		fmt.Fprint(os.Stderr, "Two-factor authentication code: ")
	}
	return readSecret()
}

// promptSMSCode prompts for the SMS PIN sent to the user's registered phone.
func promptSMSCode(_ context.Context, _ api.SMSChallenge) (string, error) {
	fmt.Fprint(os.Stderr, "SMS verification code: ")
	return readSecret()
}

// promptNewPassword is called when the server forces a password change. It
// collects the user's email address and new password.
func promptNewPassword(_ context.Context, challenge api.PasswordChangeChallenge) (string, string, error) {
	if challenge.Email != "" {
		fmt.Fprintf(os.Stderr, "Email address [%s]: ", challenge.Email)
	} else {
		fmt.Fprint(os.Stderr, "Email address: ")
	}

	email := challenge.Email
	sc := bufio.NewScanner(os.Stdin)
	if sc.Scan() {
		if v := strings.TrimSpace(sc.Text()); v != "" {
			email = v
		}
	}

	fmt.Fprint(os.Stderr, "New password: ")
	password, err := readSecret()
	if err != nil {
		return "", "", fmt.Errorf("reading new password: %w", err)
	}

	fmt.Fprint(os.Stderr, "Confirm new password: ")
	confirm, err := readSecret()
	if err != nil {
		return "", "", fmt.Errorf("reading confirmation: %w", err)
	}
	if password != confirm {
		return "", "", fmt.Errorf("passwords do not match")
	}

	return email, password, nil
}

// readSecret reads one line from stdin. Input is hidden when stdin is a real
// TTY; falls back to a plain line read when piped.
func readSecret() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // move to next line after hidden input
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(raw)), nil
	}
	sc := bufio.NewScanner(os.Stdin)
	if sc.Scan() {
		return strings.TrimSpace(sc.Text()), nil
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no input provided")
}

// ---------------------------------------------------------------------------
// Help text
// ---------------------------------------------------------------------------

func printUsage() {
	fmt.Fprint(os.Stderr, `cli-login-preference - demonstrates LoginMethod / LoginPreference selection

USAGE
  cli-login-preference [flags]

CONNECTION FLAGS
  -host   string     Cumulocity base URL, e.g. https://mytenant.cumulocity.com.
                     Overrides $C8Y_BASEURL / $C8Y_URL / $C8Y_HOST.
                     Prompted interactively when required and not provided.

  -user   string     Username. Overrides $C8Y_USER / $C8Y_USERNAME.
                     Prompted interactively when required and not provided.

  -password string   Password. Overrides $C8Y_PASSWORD.
                     Prompted interactively when required and not provided.

  -tenant string     Tenant ID (optional). Overrides $C8Y_TENANT.

  -no-env            Do not read any values from environment variables.
                     All connection details must be supplied via flags.

  -debug             Enable debug mode with full HTTP request/response logging.
                     Sensitive headers (e.g. Authorization) are shown unredacted.

AUTHENTICATION FLAGS
  -method  string    Enforce exactly one login method (strict).
                     Fails immediately when required credentials are absent
                     or the tenant has no external OAuth2 provider.
                     Values: BASIC | OAUTH2_INTERNAL | CERTIFICATE |
                             OAUTH2_DEVICE_FLOW | OAUTH2_BROWSER_FLOW

  -prefer  string    Add a method to the ordered preference list (repeatable).
                     Methods are tried in order; unavailable ones are skipped.
                     -method and -prefer are mutually exclusive.
                     Example: -prefer OAUTH2_INTERNAL -prefer OAUTH2_DEVICE_FLOW

  -browser-addr      Callback URL for OAUTH2_BROWSER_FLOW, e.g.:
                       http://127.0.0.1:5001/callback  (explicit)
                       127.0.0.1:5001/login            (scheme inferred)
                       127.0.0.1:5001                  (path defaults to /callback)
                     Default: 127.0.0.1:5001

  -browser-pkce      Enable PKCE (RFC 7636) for OAUTH2_BROWSER_FLOW.
                     Requires support from both the external IdP and
                     Cumulocity's token endpoint (may not be supported yet).

  -totp-secret       Base32 TOTP secret for automatic code generation.
                     Falls back to $C8Y_TOTP_SECRET unless -no-env is set.

ENVIRONMENT VARIABLES
  C8Y_BASEURL        Tenant base URL, e.g. https://mytenant.cumulocity.com
  C8Y_USER           Username
  C8Y_PASSWORD       Password
  C8Y_TENANT         Tenant ID (optional, e.g. t12345)
  C8Y_TOKEN          Bearer token (skips username/password login)
  C8Y_TOTP_SECRET    Base32 TOTP secret (automation only)

EXAMPLES
  # Force Basic auth
  cli-login-preference -host https://mytenant.cumulocity.com -method BASIC

  # Force OAI-Secure, ignoring all environment variables
  cli-login-preference -host https://mytenant.cumulocity.com -no-env -method OAUTH2_INTERNAL

  # Open system browser for SSO authorization code flow
  cli-login-preference -method OAUTH2_BROWSER_FLOW -browser-addr localhost:8080

  # Prefer internal SSO; fall back to device flow when credentials are absent
  cli-login-preference -prefer OAUTH2_INTERNAL -prefer OAUTH2_DEVICE_FLOW

  # Device flow first; Basic auth as last resort
  cli-login-preference -prefer OAUTH2_DEVICE_FLOW -prefer BASIC

`)
}
