package password

import (
	"regexp"
	"strings"
	"testing"
)

// TestNewRandomPassword_Default tests password generation with default settings.
func TestNewRandomPassword_Default(t *testing.T) {
	pwd, err := NewRandomPassword()
	if err != nil {
		t.Fatalf("NewRandomPassword with default settings failed: %v", err)
	}

	// Default length is 31
	if len(pwd) != 31 {
		t.Errorf("Expected default password length to be 31, got %d", len(pwd))
	}

	// Check for presence of all default character types (at least one of each)
	// These checks are for presence, not exact counts, as the remaining characters
	// are filled randomly.
	if !strings.ContainsAny(pwd, lowercaseChars) {
		t.Errorf("Default password does not contain lowercase characters")
	}
	if !strings.ContainsAny(pwd, uppercaseChars) {
		t.Errorf("Default password does not contain uppercase characters")
	}
	if !strings.ContainsAny(pwd, digitChars) {
		t.Errorf("Default password does not contain digits")
	}
	if !strings.ContainsAny(pwd, symbolChars) {
		t.Errorf("Default password does not contain symbols")
	}
}

// TestNewRandomPassword_WithLength tests setting a custom password length.
func TestNewRandomPassword_WithLength(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"Length 8", 8}, // Minimum valid length
		{"Length 25", 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pwd, err := NewRandomPassword(WithLength(tt.length))
			if err != nil {
				t.Fatalf("NewRandomPassword with length %d failed: %v", tt.length, err)
			}
			if len(pwd) != tt.length {
				t.Errorf("Expected password length to be %d, got %d", tt.length, len(pwd))
			}
		})
	}
}

// TestNewRandomPassword_SpecificCounts tests generating passwords with specific minimum character counts.
func TestNewRandomPassword_SpecificCounts(t *testing.T) {
	tests := []struct {
		name         string
		length       int
		symbols      int
		digits       int
		uppercase    int
		lowercase    int
		expectedMin  map[string]int // Map to store expected minimum counts for validation
		charRegexMap map[string]*regexp.Regexp
	}{
		{
			"All Types Min", 20, 3, 4, 5, 8,
			map[string]int{"symbols": 3, "digits": 4, "uppercase": 5, "lowercase": 8},
			map[string]*regexp.Regexp{
				"symbols":   regexp.MustCompile("[" + `!@#$%^&*()-_=+\[\]{}|;:,.<>/?~` + "`" + "]"),
				"digits":    regexp.MustCompile(`[0-9]`),
				"uppercase": regexp.MustCompile(`[A-Z]`),
				"lowercase": regexp.MustCompile(`[a-z]`),
			},
		},
		{
			"Only Digits", 8, 0, 8, 0, 0,
			map[string]int{"digits": 8},
			map[string]*regexp.Regexp{
				"digits": regexp.MustCompile(`[0-9]`),
			},
		},
		{
			"Only Symbols", 10, 10, 0, 0, 0,
			map[string]int{"symbols": 10},
			map[string]*regexp.Regexp{
				"symbols": regexp.MustCompile("[" + `!@#$%^&*()-_=+\[\]{}|;:,.<>/?~` + "`" + "]"),
			},
		},
		{
			"Only Uppercase", 12, 0, 0, 12, 0,
			map[string]int{"uppercase": 12},
			map[string]*regexp.Regexp{
				"uppercase": regexp.MustCompile(`[A-Z]`),
			},
		},
		{
			"Only Lowercase", 8, 0, 0, 0, 8,
			map[string]int{"lowercase": 8},
			map[string]*regexp.Regexp{
				"lowercase": regexp.MustCompile(`[a-z]`),
			},
		},
		{
			"Mixed No Symbols", 15, 0, 3, 4, 8,
			map[string]int{"digits": 3, "uppercase": 4, "lowercase": 8},
			map[string]*regexp.Regexp{
				"digits":    regexp.MustCompile(`[0-9]`),
				"uppercase": regexp.MustCompile(`[A-Z]`),
				"lowercase": regexp.MustCompile(`[a-z]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pwd, err := NewRandomPassword(
				WithLength(tt.length),
				WithSymbols(tt.symbols),
				WithDigits(tt.digits),
				WithUppercase(tt.uppercase),
				WithLowercase(tt.lowercase),
			)
			if err != nil {
				t.Fatalf("NewRandomPassword for %s failed: %v", tt.name, err)
			}

			if len(pwd) != tt.length {
				t.Errorf("Expected password length %d, got %d for %s", tt.length, len(pwd), tt.name)
			}

			// Count actual character types in the generated password
			actualCounts := map[string]int{
				"symbols":   0,
				"digits":    0,
				"uppercase": 0,
				"lowercase": 0,
			}

			for _, char := range pwd {
				s := string(char)
				if strings.ContainsAny(s, symbolChars) {
					actualCounts["symbols"]++
				} else if strings.ContainsAny(s, digitChars) {
					actualCounts["digits"]++
				} else if strings.ContainsAny(s, uppercaseChars) {
					actualCounts["uppercase"]++
				} else if strings.ContainsAny(s, lowercaseChars) {
					actualCounts["lowercase"]++
				}
			}

			// Validate minimum counts
			for charType, minCount := range tt.expectedMin {
				if actualCounts[charType] < minCount {
					t.Errorf("%s: Expected at least %d %s, got %d", tt.name, minCount, charType, actualCounts[charType])
				}
			}

			// Ensure only expected character types are present (if specific counts were set)
			if tt.symbols == 0 && actualCounts["symbols"] > 0 {
				t.Errorf("%s: Expected no symbols, but found %d", tt.name, actualCounts["symbols"])
			}
			if tt.digits == 0 && actualCounts["digits"] > 0 {
				t.Errorf("%s: Expected no digits, but found %d", tt.name, actualCounts["digits"])
			}
			if tt.uppercase == 0 && actualCounts["uppercase"] > 0 {
				t.Errorf("%s: Expected no uppercase, but found %d", tt.name, actualCounts["uppercase"])
			}
			if tt.lowercase == 0 && actualCounts["lowercase"] > 0 {
				t.Errorf("%s: Expected no lowercase, but found %d", tt.name, actualCounts["lowercase"])
			}
		})
	}
}

// TestNewRandomPassword_ErrorCases tests scenarios that should result in an error.
func TestNewRandomPassword_ErrorCases(t *testing.T) {
	tests := []struct {
		name string
		opts []PasswordOption
	}{
		{
			"Required Chars Exceeds Length",
			[]PasswordOption{
				WithLength(5),
				WithSymbols(2),
				WithDigits(2),
				WithUppercase(2),
			},
		},
		{
			"Zero Length",
			[]PasswordOption{WithLength(0)},
		},
		{
			"Negative Length",
			[]PasswordOption{WithLength(-5)},
		},
		{
			"No Character Types Enabled", // When length > 0 but all min counts are 0 and no default is applied
			[]PasswordOption{
				WithLength(10),
				WithSymbols(0),
				WithDigits(0),
				WithUppercase(0),
				WithLowercase(0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRandomPassword(tt.opts...)
			if err == nil {
				t.Errorf("Expected an error for %s, but got none", tt.name)
			}
		})
	}
}

// TestShuffleBytes verifies that shuffleBytes shuffles the slice.
// This is a probabilistic test, so it's not guaranteed to fail if shuffle is broken,
// but it's a good sanity check.
func TestShuffleBytes(t *testing.T) {
	original := []byte("abcdefg")
	shuffled := make([]byte, len(original))
	copy(shuffled, original)

	shuffleBytes(shuffled)

	// Check if the content is the same (i.e., no characters were lost/added)
	if len(original) != len(shuffled) {
		t.Errorf("Shuffled slice length changed. Original: %d, Shuffled: %d", len(original), len(shuffled))
	}

	// Check if the sorted content is the same
	sortedOriginal := string(original)
	// Sort the shuffled slice to compare content, not order
	// This is a common way to check if all elements are preserved.
	// We can't use strings.Sort, so convert to rune slice, sort, convert back.
	runeShuffled := []rune(string(shuffled))
	for i := 0; i < len(runeShuffled)-1; i++ {
		for j := i + 1; j < len(runeShuffled); j++ {
			if runeShuffled[i] > runeShuffled[j] {
				runeShuffled[i], runeShuffled[j] = runeShuffled[j], runeShuffled[i]
			}
		}
	}
	if string(runeShuffled) != sortedOriginal {
		t.Errorf("Shuffled slice content changed. Original: %s, Shuffled: %s", sortedOriginal, string(runeShuffled))
	}

	// This is a weak check for randomness, but better than nothing:
	// Check if it's highly unlikely to be the same as the original (after a few runs).
	// For a truly robust test, one might use statistical tests or run many iterations.
	if string(original) == string(shuffled) {
		t.Log("Warning: Shuffled slice is identical to original. This is statistically unlikely but possible.")
	}
}

// TestGetRandomChar ensures getRandomChar returns a character from the given charset.
func TestGetRandomChar(t *testing.T) {
	charset := "abc"
	for i := 0; i < 100; i++ { // Run multiple times to increase confidence
		char, err := getRandomChar(charset)
		if err != nil {
			t.Fatalf("getRandomChar failed: %v", err)
		}
		if !strings.ContainsRune(charset, rune(char)) {
			t.Errorf("getRandomChar returned '%c' which is not in charset '%s'", char, charset)
		}
	}

	// Test with empty charset
	_, err := getRandomChar("")
	if err == nil {
		t.Error("Expected error for empty charset, got none")
	}
	if err != nil && !strings.Contains(err.Error(), "charset is empty") {
		t.Errorf("Expected 'charset is empty' error, got: %v", err)
	}
}
