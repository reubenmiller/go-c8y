// Package main demonstrates a full Cumulocity username/password login that
// transparently handles TOTP enrollment, TOTP login challenges, SMS TFA
// challenges, and forced password changes.
//
// The login flows are driven by callbacks supplied via api.LoginOptions, keeping
// the library free of terminal-UI dependencies:
//
//   - TOTPCode      – called whenever a TOTP code must be entered
//   - QRCode        – called with the otpauth:// URL during first-time enrollment
//   - SMSCode       – called when the server sends an SMS PIN challenge
//   - PasswordChange – called when the server forces a password reset on login
//
// After a successful login the example verifies the token with a lightweight
// API call and lists the first page of devices.
//
// Required environment variables:
//
//	C8Y_BASEURL     – e.g. https://mytenant.cumulocity.com
//	C8Y_USER        – Cumulocity username
//	C8Y_PASSWORD    – Cumulocity password
//	C8Y_TENANT      – (optional) tenant ID (e.g. t12345)
//	C8Y_TOTP_SECRET – (optional) base32 TOTP secret passed to LoginOptions.TOTPSecret
//	                  for automatic code generation. Intended for machine/automation
//	                  scenarios only; NOT recommended for interactive users.
package main

import (
	"bufio"
	"context"
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
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users/currentuser/totp"
	"golang.org/x/term"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx := context.Background()

	// -----------------------------------------------------------------------
	// 1. Build the client — credentials come from the environment.
	// -----------------------------------------------------------------------
	client := api.NewClient(api.ClientOptions{
		BaseURL: authentication.HostFromEnvironment(),
		Auth:    authentication.FromEnvironment(),
	})

	// -----------------------------------------------------------------------
	// 2. Login with full TFA/password-change handling via callbacks.
	// -----------------------------------------------------------------------
	_, err := client.LoginWithOptions(ctx, api.LoginOptions{
		// TOTPSecret enables automatic TOTP code generation for machine/automation
		// scenarios. When set, interactive TOTP prompts are skipped entirely.
		// WARNING: storing the secret alongside credentials removes the second-factor
		// security benefit; use only where truly unattended operation is required.
		TOTPSecret: os.Getenv("C8Y_TOTP_SECRET"),

		// QRCode is called once during first-time TOTP enrollment so the user
		// can scan the secret into their authenticator app.
		QRCode: func(otpauthURL string, secret string) {
			fmt.Fprintf(os.Stderr, "\n📱 Scan the QR code below with your authenticator app:\n\n")
			qrterminal.GenerateWithConfig(otpauthURL, qrterminal.Config{
				Level:      qrterminal.M,
				Writer:     os.Stderr,
				HalfBlocks: true,
				QuietZone:  1,
			})
			fmt.Fprintf(os.Stderr, "\n secret: %s\n\n", secret)
			fmt.Fprintf(os.Stderr, "\nManual entry URL: %s\n", otpauthURL)
		},

		// TOTPCode is called whenever a TOTP code is needed.
		// challenge.IsSetup is true during enrollment, false for a login challenge.
		TOTPCode: promptTOTPCode,

		// SMSCode is called when the server sends an SMS PIN to the user's phone.
		SMSCode: promptSMSCode,

		// PasswordChange is called when the server forces a password reset.
		PasswordChange: promptNewPassword,
	})
	if err != nil {
		slog.Error("Login failed", "err", err)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------
	// 3. Verify the token with a lightweight API call.
	// -----------------------------------------------------------------------
	result := client.Tenants.Current.Get(ctx, currenttenant.GetOptions{})
	if result.Err != nil {
		slog.Error("API call failed", "err", result.Err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "✅ Authenticated\n")
	fmt.Fprintf(os.Stderr, "   Tenant:  %s\n", result.Data.Name())
	fmt.Fprintf(os.Stderr, "   Domain:  %s\n", result.Data.DomainName())

	// -----------------------------------------------------------------------
	// 4. List devices as a quick smoke-test.
	// -----------------------------------------------------------------------
	devicesResult := client.Devices.List(ctx, devices.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			WithTotalElements: true,
			PageSize:          1,
		},
	})
	if devicesResult.Err != nil {
		slog.Error("Devices API call failed", "err", devicesResult.Err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "   Devices: %v\n", devicesResult.Meta["totalElements"])
}

// promptTOTPCode is a TOTPCodeFunc implementation that reads a TOTP code from
// the terminal. Input is hidden when stdin is a real TTY; if stdin is piped
// (e.g. in scripts) it falls back to a plain line read.
func promptTOTPCode(_ context.Context, challenge totp.TOTPChallenge) (string, error) {
	if challenge.IsSetup {
		fmt.Fprint(os.Stderr, "Enter the 6-digit code from your authenticator app to verify enrollment: ")
	} else {
		fmt.Fprint(os.Stderr, "Two-factor authentication code: ")
	}

	// Use golang.org/x/term so the code isn't echoed on a real terminal.
	if term.IsTerminal(int(os.Stdin.Fd())) {
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // move to next line after hidden input
		if err != nil {
			return "", fmt.Errorf("reading TOTP code: %w", err)
		}
		return strings.TrimSpace(string(raw)), nil
	}

	// Piped / non-TTY fallback.
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading TOTP code: %w", err)
	}
	return "", fmt.Errorf("no TOTP code provided")
}

// promptSMSCode is an SMSCode callback that prompts the user for the PIN sent
// to their registered phone number.
func promptSMSCode(_ context.Context, _ api.SMSChallenge) (string, error) {
	fmt.Fprint(os.Stderr, "SMS verification code: ")

	if term.IsTerminal(int(os.Stdin.Fd())) {
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("reading SMS code: %w", err)
		}
		return strings.TrimSpace(string(raw)), nil
	}

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading SMS code: %w", err)
	}
	return "", fmt.Errorf("no SMS code provided")
}

// promptNewPassword is a PasswordChange callback that prompts the user for
// their email and a new password when the server forces a password change.
func promptNewPassword(_ context.Context, challenge api.PasswordChangeChallenge) (string, string, error) {
	// Prompt for the email address. The username is provided as a hint but may
	// not be the user's actual email.
	var email string
	if term.IsTerminal(int(os.Stdin.Fd())) {
		if challenge.Email != "" {
			fmt.Fprintf(os.Stderr, "Email address [%s]: ", challenge.Email)
		} else {
			fmt.Fprint(os.Stderr, "Email address: ")
		}
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			email = strings.TrimSpace(scanner.Text())
		}
		if email == "" {
			email = challenge.Email
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			email = strings.TrimSpace(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return "", "", fmt.Errorf("reading email: %w", err)
		}
	}
	if email == "" {
		return "", "", fmt.Errorf("no email provided")
	}

	for {
		fmt.Fprint(os.Stderr, "Your password has expired. Enter a new password: ")

		var pw string
		if term.IsTerminal(int(os.Stdin.Fd())) {
			raw, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return "", "", fmt.Errorf("reading new password: %w", err)
			}
			pw = strings.TrimSpace(string(raw))
		} else {
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					return "", "", fmt.Errorf("reading new password: %w", err)
				}
				return "", "", fmt.Errorf("no new password provided")
			}
			pw = strings.TrimSpace(scanner.Text())
		}

		strength := users.CalculatePasswordStrength(pw)
		if strength == users.PasswordStrengthRed {
			fmt.Fprintf(os.Stderr, "Password is too weak (must be at least 8 characters with uppercase, lowercase, digits and symbols). Please try again.\n")
			continue
		}
		if strength == users.PasswordStrengthYellow {
			fmt.Fprintf(os.Stderr, "Warning: password strength is moderate. Consider using a stronger password (add more character types).\n")
		}

		if term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Fprint(os.Stderr, "Confirm new password: ")
			confirm, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return "", "", fmt.Errorf("reading password confirmation: %w", err)
			}
			if pw != strings.TrimSpace(string(confirm)) {
				fmt.Fprintf(os.Stderr, "Passwords do not match. Please try again.\n")
				continue
			}
		}

		return email, pw, nil
	}
}
