package notification2

import (
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
)

func Test_ParseMessage(t *testing.T) {
	raw := []byte(`CLJuEJgjIAAwAQ==
/t123456/measurements/12345
CREATE

{"self":"https://example.com/measurement/measurements/12345","time":"2024-10-02T12:11:00.000Z","id":"12345","source":{"self":"https://example.com/inventory/managedObjects/11111","id":"11111"},"type":"temperature"}`)

	message := parseMessage(raw)

	testingutils.Equals(t, "CLJuEJgjIAAwAQ==", string(message.Identifier))
	testingutils.Equals(t, "CREATE", string(message.Action))
	testingutils.Equals(t, "/t123456/measurements/12345", string(message.Description))
	testingutils.Assert(t, len(message.Payload) > 0, "payload size should be larger than zero")
}

func Test_GetEndpoint(t *testing.T) {
	endpoint := getEndpoint("http://cumulocity:8111", Subscription{
		Token:    "abcdef",
		Consumer: "",
	})
	testingutils.Equals(t, endpoint.Scheme, "ws")

	endpoint = getEndpoint("https://cumulocity:8111", Subscription{
		Token:    "abcdef",
		Consumer: "",
	})
	testingutils.Equals(t, endpoint.Scheme, "wss")
}
