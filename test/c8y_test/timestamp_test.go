package c8y_test

import (
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func TestTimestamp_UnmarshalJSON_RFC3339Nano(t *testing.T) {

	timestampStr := "2019-04-06T14:11:42.045421+02:00"

	// Add quotes around it to simulate json value
	timestampBytes := []byte(`"` + timestampStr + `"`)

	c8yTimestamp := c8y.Timestamp{}
	err := c8yTimestamp.UnmarshalJSON(timestampBytes)
	testingutils.Ok(t, err)
	testingutils.Equals(t, timestampStr, c8yTimestamp.String())
}

func TestTimestamp_UnmarshalJSON_RFC3339(t *testing.T) {
	timestampStr := "2019-04-06T14:11:42+02:00"

	// Add quotes around it to simulate json value
	timestampBytes := []byte(`"` + timestampStr + `"`)

	c8yTimestamp := c8y.Timestamp{}
	err := c8yTimestamp.UnmarshalJSON(timestampBytes)
	testingutils.Ok(t, err)
	testingutils.Equals(t, timestampStr, c8yTimestamp.String())
}
