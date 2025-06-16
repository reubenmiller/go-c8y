package c8y_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func Test_BulkRegistrationRecordCollection(t *testing.T) {
	csvContents := &strings.Builder{}
	csvErr := c8y.BulkRegistrationRecordWriter(
		csvContents,
		c8y.BulkRegistrationRecord{
			ID:            "device01",
			AuthType:      c8y.BulkRegistrationAuthTypeCertificates,
			EnrollmentOTP: "dummy",
			Name:          "device01",
			Type:          "linux",
			IDType:        "c8y_Serial",
			IsAgent:       true,
		},
		c8y.BulkRegistrationRecord{
			ID:       "device02",
			AuthType: c8y.BulkRegistrationAuthTypeCertificates,
			Name:     "device02",
			Type:     "macOS",
			IDType:   "c8y_Serial",
			IsAgent:  true,
		},
		c8y.BulkRegistrationRecord{
			ID:          "device03",
			AuthType:    c8y.BulkRegistrationAuthTypeBasic,
			Credentials: "pass12345",
			Name:        "device03",
			Type:        "Windows",
			IDType:      "c8y_Serial",
			IsAgent:     false,
		},
	)
	testingutils.Ok(t, csvErr)
	fmt.Printf("%s\n", csvContents.String())
	expected := strings.TrimLeft(`
ID	AUTH_TYPE	CREDENTIALS	ENROLLMENT_OTP	NAME	TYPE	IDTYPE	ICCID	TENANT	PATH	com_cumulocity_model_Agent.active
device01	CERTIFICATES		dummy	device01	linux	c8y_Serial				true
device02	CERTIFICATES			device02	macOS	c8y_Serial				true
device03	BASIC	pass12345		device03	Windows	c8y_Serial				false
`, "\n")
	testingutils.Equals(t, expected, csvContents.String())
}
