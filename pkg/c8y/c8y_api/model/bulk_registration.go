package model

import (
	"encoding/csv"
	"fmt"
	"io"
)

// BulkNewDeviceRequest response which details the results of the bulk registration
type BulkNewDeviceRequest struct {
	NumberOfAll        int64 `json:"numberOfAll,omitempty"`
	NumberOfCreated    int64 `json:"numberOfCreated,omitempty"`
	NumberOfFailed     int64 `json:"numberOfFailed,omitempty"`
	NumberOfSuccessful int64 `json:"numberOfSuccessful,omitempty"`

	CredentialUpdatedList []BulkNewDeviceRequestDetails `json:"credentialUpdatedList,omitempty"`
	FailedCreationList    []BulkNewDeviceRequestDetails `json:"failedCreationList,omitempty"`
}

type BulkNewDeviceRequestDetails struct {
	BulkNewDeviceStatus string `json:"bulkNewDeviceStatus,omitempty"`
	DeviceID            string `json:"deviceId,omitempty"`

	FailureReason string `json:"failureReason,omitempty"`
	Line          string `json:"line,omitempty"`
}

type BulkRegistrationAuthType string

const (
	// BulkRegistrationAuthTypeBasic Basic Authorization
	BulkRegistrationAuthTypeBasic BulkRegistrationAuthType = "BASIC"

	// BulkRegistrationAuthTypeCertificates Certificate Authorization
	BulkRegistrationAuthTypeCertificates BulkRegistrationAuthType = "CERTIFICATES"
)

type BulkRegistrationRecord struct {
	// External ID
	ID string `json:"externalId,omitempty"`

	// External Id Type
	IDType string `json:"externalType,omitempty"`

	// Authorization Type, BASIC, CERTIFICATES
	AuthType BulkRegistrationAuthType `json:"authType,omitempty"`

	// Basic Auth credentials
	Credentials string `json:"password,omitempty"`

	// Enrollment one-time password
	EnrollmentOTP string `json:"enrollmentOTP,omitempty"`

	// Name
	Name string `json:"name,omitempty"`

	// Type
	Type string `json:"type,omitempty"`

	// ICCID
	ICCID string `json:"iccid,omitempty"`

	// Tenant
	Tenant string `json:"tenant,omitempty"`

	// Path / Group hierarchy
	Path string `json:"group,omitempty"`

	// Is the device an agent
	IsAgent bool `json:"isAgent,omitempty"`
}

// SetBasicAuth sets the record for basic authentication
func (r *BulkRegistrationRecord) SetBasicAuth(v string) {
	r.AuthType = BulkRegistrationAuthTypeBasic
	r.Credentials = v
	r.IsAgent = true
}

// SetCertificateAuth set certificate authentication (for externally issued certificates)
func (r *BulkRegistrationRecord) SetCertificateAuth() {
	r.AuthType = BulkRegistrationAuthTypeCertificates
	r.Credentials = ""
	r.IsAgent = true
}

// SetEnrollmentPassword set certificate authentication with a one-time password for the Cumulocity certificate authority
func (r *BulkRegistrationRecord) SetEnrollmentPassword(v string) {
	r.AuthType = BulkRegistrationAuthTypeCertificates
	r.EnrollmentOTP = v
	r.Credentials = ""
	r.IsAgent = true
}

// BulkRegistrationColumns bulk registration CSV columns
var BulkRegistrationColumns = []string{
	"ID",          // External ID
	"AUTH_TYPE",   // Authorization type
	"CREDENTIALS", // Basic Auth
	"NAME",        // Device name
	"TYPE",        // Device type
	"IDTYPE",      // External ID Type
	"ICCID",       // ICCID
	"TENANT",      // Tenant
	"PATH",        // Path / group
	"com_cumulocity_model_Agent.active",
}

// WriteCSV the records to the give writer
func BulkRegistrationRecordWriter(w io.Writer, records ...BulkRegistrationRecord) error {
	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = '\t'
	if err := csvWriter.Write(BulkRegistrationColumns); err != nil {
		return err
	}
	for _, item := range records {
		if err := csvWriter.Write([]string{
			item.ID,
			string(item.AuthType),
			item.Credentials,
			item.Name,
			item.Type,
			item.IDType,
			item.ICCID,
			item.Tenant,
			item.Path,
			fmt.Sprintf("%v", item.IsAgent),
		}); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

// BulkRegistrationCertificateAuthorityColumns bulk registration CSV columns
var BulkRegistrationCertificateAuthorityColumns = []string{
	"ID",             // External ID
	"AUTH_TYPE",      // Authorization type
	"CREDENTIALS",    // Basic Auth
	"ENROLLMENT_OTP", // One-time password for EST enrollment
	"NAME",           // Device name
	"TYPE",           // Device type
	"IDTYPE",         // External ID Type
	"ICCID",          // ICCID
	"TENANT",         // Tenant
	"PATH",           // Path / group
	"com_cumulocity_model_Agent.active",
}

func BulkRegistrationCertificateAuthorityWriter(w io.Writer, records ...BulkRegistrationRecord) error {
	csvWriter := csv.NewWriter(w)
	csvWriter.Comma = '\t'
	if err := csvWriter.Write(BulkRegistrationCertificateAuthorityColumns); err != nil {
		return err
	}
	for _, item := range records {
		if err := csvWriter.Write([]string{
			item.ID,
			string(item.AuthType),
			item.Credentials,
			item.EnrollmentOTP,
			item.Name,
			item.Type,
			item.IDType,
			item.ICCID,
			item.Tenant,
			item.Path,
			fmt.Sprintf("%v", item.IsAgent),
		}); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}
