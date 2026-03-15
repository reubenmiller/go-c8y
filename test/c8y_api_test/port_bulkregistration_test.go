package api_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_BulkRegistrationRecordCollectionMixed(t *testing.T) {

	csvContents := &strings.Builder{}
	csvErr := model.BulkRegistrationRecordWriter(
		csvContents,
		model.BulkRegistrationRecord{
			ID:       "device02",
			AuthType: model.BulkRegistrationAuthTypeCertificates,
			Name:     "device02",
			Type:     "macOS",
			IDType:   "c8y_Serial",
			IsAgent:  true,
		},
		model.BulkRegistrationRecord{
			ID:          "device03",
			AuthType:    model.BulkRegistrationAuthTypeBasic,
			Credentials: "pass12345",
			Name:        "device03",
			Type:        "Windows",
			IDType:      "c8y_Serial",
			IsAgent:     false,
		},
	)
	require.NoError(t, csvErr)
	fmt.Printf("%s\n", csvContents.String())
	expected := strings.TrimLeft(`
ID	AUTH_TYPE	CREDENTIALS	NAME	TYPE	IDTYPE	ICCID	TENANT	PATH	com_cumulocity_model_Agent.active
device02	CERTIFICATES		device02	macOS	c8y_Serial				true
device03	BASIC	pass12345	device03	Windows	c8y_Serial				false
`, "\n")
	assert.Equal(t, expected, csvContents.String())
}

func Test_BulkRegistrationRecordCollectionWithOneTimePassword(t *testing.T) {

	csvContents := &strings.Builder{}
	csvErr := model.BulkRegistrationRecordWriter(
		csvContents,
		model.BulkRegistrationRecord{
			ID:            "device01",
			AuthType:      model.BulkRegistrationAuthTypeCertificates,
			EnrollmentOTP: "dummy",
			Name:          "device01",
			Type:          "linux",
			IDType:        "c8y_Serial",
			IsAgent:       true,
		},
	)
	require.NoError(t, csvErr)
	fmt.Printf("%s\n", csvContents.String())
	expected := strings.TrimLeft(`
ID	AUTH_TYPE	CREDENTIALS	NAME	TYPE	IDTYPE	ICCID	TENANT	PATH	com_cumulocity_model_Agent.active
device01	CERTIFICATES		device01	linux	c8y_Serial				true
`, "\n")
	assert.Equal(t, expected, csvContents.String())
}
