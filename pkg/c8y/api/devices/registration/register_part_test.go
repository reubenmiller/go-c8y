package registration

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestRegisterSendsPlainFilePart verifies the CSV of Register is the plain
// "file" form field: no file name and no content type, as go-c8y-cli v2 sent.
func TestRegisterSendsPlainFilePart(t *testing.T) {
	var disposition, contentType, content string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			reader, err := r.MultipartReader()
			if err != nil {
				t.Errorf("multipart: %v", err)
				return
			}
			part, err := reader.NextPart()
			if err != nil {
				t.Errorf("part: %v", err)
				return
			}
			disposition = part.Header.Get("Content-Disposition")
			contentType = part.Header.Get("Content-Type")
			b, _ := io.ReadAll(part)
			content = string(b)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"numberOfAll":1,"numberOfCreated":1,"numberOfFailed":0,"numberOfSuccessful":1,"status":"CREATED"}`))
			return
		}
		_, _ = w.Write([]byte(`{"externalId":"tmp01","type":"c8y_Serial","managedObject":{"id":"12"}}`))
	})
	defer closeFn()

	res := svc.Register(context.Background(), RegisterOptions{ExternalID: "tmp01", AuthType: AuthTypeCertificates, EnrollmentOTP: "abc"})
	if res.Err != nil {
		t.Fatalf("Register: %v", res.Err)
	}
	if disposition != `form-data; name="file"` {
		t.Errorf("Content-Disposition = %q, want the part without a file name", disposition)
	}
	if contentType != "" {
		t.Errorf("Content-Type = %q, want none", contentType)
	}
	if !strings.HasPrefix(content, "ID\tAUTH_TYPE\tENROLLMENT_OTP") {
		t.Errorf("file = %q", content)
	}
}
