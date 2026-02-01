package trustedcertificates

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/trustedcertificates/certificateauthority"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/trustedcertificates/certificaterevocationlist"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
	"resty.dev/v3"
)

var ApiTrustedCertificates = "/tenant/tenants/{tenantID}/trusted-certificates"
var ApiTrustedCertificate = "/tenant/tenants/{tenantID}/trusted-certificates/{fingerprint}"

// Proof of possession
var ApiProofOfPossession = "/tenant/tenants/{tenantID}/trusted-certificates-pop/{fingerprint}/pop"
var ApiGenerateVerificationCode = "/tenant/tenants/{tenantID}/trusted-certificates-pop/{fingerprint}/verification-code"
var ApiProofOfPossessionConfirm = "/tenant/tenants/{tenantID}/trusted-certificates-pop/{fingerprint}/confirmed"

const ParamTenant = "tenantID"
const ParamFingerprint = "fingerprint"

const ResultProperty = "certificates"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:              *s,
		CertificateAuthority: certificateauthority.NewService(s),
		RevocationList:       certificaterevocationlist.NewService(s),
	}
}

// Service api to interact with the trusted certificates
// type Service core.Service
type Service struct {
	core.Service

	CertificateAuthority *certificateauthority.Service
	RevocationList       *certificaterevocationlist.Service
}

// ListOptions trusted certificates filter options
type ListOptions struct {
	TenantID string

	// When set to true, the tenant certificate authority will be retrieved
	CertificateAuthority bool `url:"certificateAuthority,omitempty"`

	// Pagination options
	pagination.PaginationOptions
}

// TrustedCertificateIterator provides iteration over trusted certificates
type TrustedCertificateIterator = pagination.Iterator[jsonmodels.TrustedCertificate]

// List trusted certificates
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.TrustedCertificate] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewTrustedCertificate)
}

// ListAll returns an iterator for all trusted certificates
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *TrustedCertificateIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return pagination.Paginate(ctx, func(page int) op.Result[jsonmodels.TrustedCertificate] {
		opts.CurrentPage = page
		return s.List(ctx, opts)
	}, jsonmodels.NewTrustedCertificate, opts.GetMaxItems())
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamTenant, opt.TenantID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTrustedCertificates)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type CreateOptions struct {
	TenantID string

	// If set to true the certificate is added to the truststore
	// The truststore contains all trusted certificates. A connection to a device is only established if it connects to Cumulocity with a certificate in the truststore.
	// Default: true
	AddToTrustStore *bool `url:"addToTrustStore,omitempty"`
}

// Create a trusted certificate
func (s *Service) Create(ctx context.Context, opt CreateOptions, body any) op.Result[jsonmodels.TrustedCertificate] {
	return core.ExecuteReturnResult(ctx, s.CreateB(opt, body), jsonmodels.NewTrustedCertificate)
}

func (s *Service) CreateB(opt CreateOptions, body any) *core.TryRequest {
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamTenant, opt.TenantID).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTrustedCertificates)
	return core.NewTryRequest(s.Client, req)
}

// Create multiple trusted certificate
func (s *Service) CreateMultiple(ctx context.Context, opt CreateOptions, body any) op.Result[jsonmodels.TrustedCertificate] {
	return core.ExecuteReturnCollection(ctx, s.CreateMultipleB(opt, body), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewTrustedCertificate)
}

func (s *Service) CreateMultipleB(opt CreateOptions, body any) *core.TryRequest {
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamTenant, opt.TenantID).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTrustedCertificates)
	return core.NewTryRequest(s.Client, req)
}

type GetOptions struct {
	TenantID string

	Fingerprint string
}

// Get a trusted certificate
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.TrustedCertificate] {
	return core.ExecuteReturnResult(ctx, s.GetB(opt), jsonmodels.NewTrustedCertificate)
}

func (s *Service) GetB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamTenant, opt.TenantID).
		SetPathParam(ParamFingerprint, opt.Fingerprint).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTrustedCertificate)
	return core.NewTryRequest(s.Client, req)
}

type UpdateOptions struct {
	TenantID string

	Fingerprint string
}

// Update a trusted certificate
func (s *Service) Update(ctx context.Context, opt UpdateOptions, body any) op.Result[jsonmodels.TrustedCertificate] {
	return core.ExecuteReturnResult(ctx, s.UpdateB(opt, body), jsonmodels.NewTrustedCertificate)
}

func (s *Service) UpdateB(opt UpdateOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamTenant, opt.TenantID).
		SetPathParam(ParamFingerprint, opt.Fingerprint).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTrustedCertificate)
	return core.NewTryRequest(s.Client, req)
}

// DeleteOptions options to delete a tenant
type DeleteOptions struct {
	TenantID string `url:"-"`

	Fingerprint string `url:"-"`
}

// Delete a trusted certificate
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(opt))
}

func (s *Service) DeleteB(opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamTenant, opt.TenantID).
		SetPathParam(ParamFingerprint, opt.Fingerprint).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiTrustedCertificate)
	return core.NewTryRequest(s.Client, req)
}

/*
  Proof of Possession
*/
// ProofOptions proof of possession verification options
type ProofOptions struct {
	Fingerprint string `url:"-"`

	TenantID string

	// Verification code. If left blank then it will be fetched
	Code string

	// Path to the private key to use to verify the code
	PrivateKey string
}

func (s *Service) ProofEndToEnd(ctx context.Context, opt ProofOptions) op.Result[jsonmodels.TrustedCertificate] {
	if opt.PrivateKey != "" {
		key, err := certutil.PrivateKeyFromFile(opt.PrivateKey)
		if err != nil {
			return op.Failed[jsonmodels.TrustedCertificate](fmt.Errorf("Failed to get private key. %w", err), false)
		}
		signer, err := certutil.NewSignerFromKey(key)
		if err != nil {
			return op.Failed[jsonmodels.TrustedCertificate](fmt.Errorf("Failed to create signer for private key. %w", err), false)
		}

		certResult := s.Get(ctx, GetOptions{
			TenantID:    opt.TenantID,
			Fingerprint: opt.Fingerprint,
		})
		if certResult.IsError() {
			return certResult
		}

		verificationCode := certResult.Data.ProofOfPossessionUnsignedVerificationCode()

		// Request a proof of possession code if the current one has expired
		if time.Now().After(certResult.Data.ProofOfPossessionVerificationCodeUsableUntil()) {
			// regeneration code
			certResult := s.CreateVerificationCode(ctx, CreateVerificationCodeOptions{
				TenantID:    opt.TenantID,
				Fingerprint: opt.Fingerprint,
			})
			if certResult.IsError() {
				return certResult
			}
			verificationCode = certResult.Data.ProofOfPossessionUnsignedVerificationCode()
		}

		code, err := signer.SignSHA256([]byte(verificationCode))
		if err != nil {
			return op.Failed[jsonmodels.TrustedCertificate](fmt.Errorf("Failed to sign verification code. %w", err), false)
		}

		opt.Code = base64.StdEncoding.EncodeToString(code)
	}
	return s.Proof(ctx, opt)
}

// Submit proof of possession
func (s *Service) Proof(ctx context.Context, opt ProofOptions) op.Result[jsonmodels.TrustedCertificate] {
	body := &model.ProofOfPossession{
		ProofOfPossessionSignedVerificationCode: opt.Code,
	}
	return core.ExecuteReturnResult(ctx, s.ProofB(opt, body), jsonmodels.NewTrustedCertificate)
}

func (s *Service) ProofB(opt ProofOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamTenant, opt.TenantID).
		SetPathParam(ParamFingerprint, opt.Fingerprint).
		SetBody(body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiProofOfPossession)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type CreateVerificationCodeOptions struct {
	Fingerprint string

	TenantID string
}

// Generate a verification code for the proof of possession operation for the certificate (by a given fingerprint)
func (s *Service) CreateVerificationCode(ctx context.Context, opt CreateVerificationCodeOptions) op.Result[jsonmodels.TrustedCertificate] {
	return core.ExecuteReturnResult(ctx, s.CreateVerificationCodeB(opt), jsonmodels.NewTrustedCertificate)
}

func (s *Service) CreateVerificationCodeB(opt CreateVerificationCodeOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamTenant, opt.TenantID).
		SetPathParam(ParamFingerprint, opt.Fingerprint).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiGenerateVerificationCode)
	return core.NewTryRequest(s.Client, req)
}

type ConfirmOptions struct {
	Fingerprint string

	TenantID string
}

// Confirm the proof of possession of an already uploaded certificate (by a given fingerprint) for a specific tenant
// TODO: This api calls always returns a 403 error
func (s *Service) Confirm(ctx context.Context, opt ConfirmOptions) op.Result[jsonmodels.TrustedCertificate] {
	return core.ExecuteReturnResult(ctx, s.ConfirmB(opt), jsonmodels.NewTrustedCertificate)
}

func (s *Service) ConfirmB(opt ConfirmOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamTenant, opt.TenantID).
		SetPathParam(ParamFingerprint, opt.Fingerprint).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiProofOfPossessionConfirm)
	return core.NewTryRequest(s.Client, req)
}
