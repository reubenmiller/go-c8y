package c8y_test

import (
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func TestTimestamp_UnmarshalJSON_RFC3339Nano(t *testing.T) {

	tstampStr := "2019-04-06T14:11:42.045421+02:00"

	// Add quotes around it to simulate json value
	tstampBytes := []byte(`"` + tstampStr + `"`)

	c8yTimestamp := c8y.Timestamp{}
	err := c8yTimestamp.UnmarshalJSON(tstampBytes)
	testingutils.Ok(t, err)
	testingutils.Equals(t, tstampStr, c8yTimestamp.String())
}

func TestTimestamp_UnmarshalJSON_RFC3339(t *testing.T) {
	tstampStr := "2019-04-06T14:11:42+02:00"

	// Add quotes around it to simulate json value
	tstampBytes := []byte(`"` + tstampStr + `"`)

	c8yTimestamp := c8y.Timestamp{}
	err := c8yTimestamp.UnmarshalJSON(tstampBytes)
	testingutils.Ok(t, err)
	testingutils.Equals(t, tstampStr, c8yTimestamp.String())
}
