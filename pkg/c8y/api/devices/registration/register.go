package registration

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/identity"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// Authentication types of a device registered with Register.
const (
	AuthTypeBasic        = "BASIC"
	AuthTypeCertificates = "CERTIFICATES"
)

// DefaultExternalIDType is the external ID type of a registered device when
// none is given, as in the Cumulocity bulk registration.
const DefaultExternalIDType = "c8y_Serial"

// RegisterOptions describes one device for Register. Empty fields are left
// out of the registration.
type RegisterOptions struct {
	// ExternalID of the device (required), e.g. its serial number
	ExternalID string

	// ExternalIDType is the type of the external ID. Defaults to
	// DefaultExternalIDType
	ExternalIDType string

	// Name of the device. Defaults to the external ID
	Name string

	// Type of the device
	Type string

	// AuthType is AuthTypeBasic (a device user with Credentials as password)
	// or AuthTypeCertificates (the device authenticates with a certificate)
	AuthType string

	// Credentials is the device user's password (AuthTypeBasic)
	Credentials string

	// EnrollmentOTP is the one-time password the device uses to request its
	// certificate from the Cumulocity certificate authority (AuthTypeCertificates)
	EnrollmentOTP string

	// ICCID of the device's SIM card, stored as c8y_Mobile.iccid
	ICCID string

	// Tenant the device is registered in (only from the management tenant)
	Tenant string

	// GroupPath is the path of the group the device is added to, with the
	// group names separated by "/". Missing groups are created
	GroupPath string

	// IsAgent marks the device as an agent
	IsAgent bool
}

// Register registers a device with the bulk registration API, which creates
// the device and its external ID right away (no device request is involved),
// and returns the device's external ID. A registration the server rejects
// (for example an external ID that exists already) is an error.
//
// In a dry run only the registration request is shown.
func (s *Service) Register(ctx context.Context, opt RegisterOptions) op.Result[jsonmodels.Identity] {
	if opt.ExternalID == "" {
		return op.Failed[jsonmodels.Identity](fmt.Errorf("registration: an external ID is required"), false)
	}
	if opt.ExternalIDType == "" {
		opt.ExternalIDType = DefaultExternalIDType
	}
	file, err := opt.csv()
	if err != nil {
		return op.Failed[jsonmodels.Identity](err, false)
	}
	result := s.CreateBulk(ctx, UploadFileOptions{Reader: bytes.NewReader(file), Name: "devices.csv"})
	if result.Err != nil || ctxhelpers.IsDryRun(ctx) {
		// Keeps the error, or the dry-run request, without any data.
		return result.Map[jsonmodels.Identity](func(jsonmodels.BulkNewDeviceRequests) jsonmodels.Identity { return jsonmodels.Identity{} })
	}
	if failures := result.Data.FailedCreationList(); result.Data.TotalFailed() > 0 || len(failures) > 0 {
		reasons := make([]string, 0, len(failures))
		for _, f := range failures {
			reasons = append(reasons, fmt.Sprintf("id=%s, reason=%s", f.DeviceID(), f.FailureReason()))
		}
		return op.Failed[jsonmodels.Identity](fmt.Errorf("registration failed: %s", strings.Join(reasons, "; ")), false)
	}
	return s.identity.Get(ctx, identity.IdentityOptions{ExternalID: opt.ExternalID, Type: opt.ExternalIDType})
}

// csv returns the bulk registration file for the device: a tab separated
// header and row with the non-empty columns.
func (o RegisterOptions) csv() ([]byte, error) {
	name := o.Name
	if name == "" {
		name = o.ExternalID
	}
	idType := o.ExternalIDType
	if idType == "" {
		idType = DefaultExternalIDType
	}
	agent := ""
	if o.IsAgent {
		agent = "true"
	}
	columns := [][2]string{
		{"ID", o.ExternalID},
		{"AUTH_TYPE", o.AuthType},
		{"CREDENTIALS", o.Credentials},
		{"ENROLLMENT_OTP", o.EnrollmentOTP},
		{"NAME", name},
		{"TYPE", o.Type},
		{"IDTYPE", idType},
		{"ICCID", o.ICCID},
		{"TENANT", o.Tenant},
		{"PATH", o.GroupPath},
		{"com_cumulocity_model_Agent.active", agent},
	}
	var header, row []string
	for _, c := range columns {
		if c[1] != "" {
			header = append(header, c[0])
			row = append(row, c[1])
		}
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	// A tab avoids quoting values that contain a comma.
	w.Comma = '\t'
	if err := w.WriteAll([][]string{header, row}); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
