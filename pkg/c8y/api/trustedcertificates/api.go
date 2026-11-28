package trustedcertificates

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/trustedcertificates/certificateauthority"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/trustedcertificates/certificaterevocationlist"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/pkg/certutil"
	"resty.dev/v3"
)

var apiTrustedCertificates = "/tenant/tenants/{tenantId}/trusted-certificates"
var apiTrustedCertificate = "/tenant/tenants/{tenantId}/trusted-certificates/{fingerprint}"

// Proof of possession
var apiProofOfPossession = "/tenant/tenants/{tenantId}/trusted-certificates-pop/{fingerprint}/pop"
var apiGenerateVerificationCode = "/tenant/tenants/{tenantId}/trusted-certificates-pop/{fingerprint}/verification-code"
var apiProofOfPossessionConfirm = "/tenant/tenants/{tenantId}/trusted-certificates-pop/{fingerprint}/confirmed"

const paramFingerprint = "fingerprint"

const resultProperty = "certificates"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc:                  *s,
		CertificateAuthority: certificateauthority.NewService(s),
		RevocationList:       certificaterevocationlist.NewService(s),
	}
}

// Service api to interact with the trusted certificates
// type Service httpcore.Service
type Service struct {
	svc                  httpcore.Service
	CertificateAuthority *certificateauthority.Service
	RevocationList       *certificaterevocationlist.Service
}

// ListOptions trusted certificates filter options
type ListOptions struct {
	TenantID string `url:"-"`

	// When set to true, the tenant certificate authority will be retrieved
	CertificateAuthority bool `url:"certificateAuthority,omitempty"`

	// Pagination options
	pagination.PaginationOptions
}

// TrustedCertificateIterator provides iteration over trusted certificates
type TrustedCertificateIterator = pagination.Iterator[jsonmodels.TrustedCertificate]

// List trusted certificates
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.TrustedCertificate] {
	return httpcore.ExecuteCollection(ctx, s.listB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewTrustedCertificate)
}

// ListAll returns an iterator for all trusted certificates
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *TrustedCertificateIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.TrustedCertificate] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewTrustedCertificate,
	)
}

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiTrustedCertificates)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

type CreateOptions struct {
	TenantID string `url:"-"`

	// If set to true the certificate is added to the truststore
	// The truststore contains all trusted certificates. A connection to a device is only established if it connects to Cumulocity with a certificate in the truststore.
	// Default: true
	AddToTrustStore *bool `url:"addToTrustStore,omitempty"`
}

// Create a trusted certificate.
// A 409 response (certificate already trusted) is treated as a duplicate (StatusDuplicate, Idempotent: true).
func (s *Service) Create(ctx context.Context, opt CreateOptions, body any) op.Result[jsonmodels.TrustedCertificate] {
	return httpcore.Execute(ctx, s.createB(opt, body), jsonmodels.NewTrustedCertificate).IgnoreConflict()
}

func (s *Service) createB(opt CreateOptions, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTrustedCertificates)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Create multiple trusted certificate
func (s *Service) CreateMultiple(ctx context.Context, opt CreateOptions, body any) op.Result[jsonmodels.TrustedCertificate] {
	return httpcore.ExecuteCollection(ctx, s.createMultipleB(opt, body), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewTrustedCertificate)
}

func (s *Service) createMultipleB(opt CreateOptions, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTrustedCertificates)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// IsFingerprint reports whether ref looks like a certificate fingerprint: a
// lowercase hex string longer than 30 characters. This mirrors the v1 CLI
// heuristic used to decide whether a reference is a fingerprint (used directly)
// or a name (looked up).
func IsFingerprint(ref string) bool {
	if len(ref) <= 30 {
		return false
	}
	for _, c := range ref {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}

// ResolveID resolves a trusted-certificate reference to its fingerprint within
// the given tenant. A fingerprint-looking value (or an explicit "id:"/
// "fingerprint:" reference) passes through unchanged with no lookup; any other
// value (or a "name:<name>" reference) is matched case-insensitively against the
// tenant's trusted certificates by name or fingerprint (first match wins),
// mirroring the v1 by-name resolver. An empty tenant defaults to the current
// tenant (handled by the client). Resolution runs for real even under dry-run /
// deferred execution so the rendered request shows the resolved fingerprint.
func (s *Service) ResolveID(ctx context.Context, tenant, ref string) (_ string, resolveErr error) {
	defer func() { resolveErr = core.WrapLookup("trusted certificate", ref, resolveErr) }()
	if ref == "" {
		return "", fmt.Errorf("empty trusted certificate reference")
	}
	if value, ok := strings.CutPrefix(ref, "id:"); ok {
		return value, nil
	}
	if value, ok := strings.CutPrefix(ref, "fingerprint:"); ok {
		return value, nil
	}
	name := ref
	if value, ok := strings.CutPrefix(ref, "name:"); ok {
		name = value
	} else if IsFingerprint(ref) {
		return ref, nil
	}

	ctx = contexthelpers.ResolutionContext(ctx)
	opt := ListOptions{TenantID: tenant}
	opt.PageSize = 100
	for cert, err := range s.ListAll(ctx, opt).Items() {
		if err != nil {
			return "", err
		}
		if strings.EqualFold(cert.Name(), name) || strings.EqualFold(cert.Fingerprint(), name) {
			return cert.Fingerprint(), nil
		}
	}
	return "", core.NewNameNotFoundError("trusted certificate", name, "trusted certificate not found: %s", name)
}

type GetOptions struct {
	TenantID string `url:"-"`

	Fingerprint string `url:"-"`
}

// Get a trusted certificate
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.TrustedCertificate] {
	return httpcore.Execute(ctx, s.getB(opt), jsonmodels.NewTrustedCertificate)
}

func (s *Service) getB(opt GetOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(paramFingerprint, opt.Fingerprint).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTrustedCertificate)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type UpdateOptions struct {
	TenantID string `url:"-"`

	Fingerprint string `url:"-"`
}

// Update a trusted certificate
func (s *Service) Update(ctx context.Context, opt UpdateOptions, body any) op.Result[jsonmodels.TrustedCertificate] {
	return httpcore.Execute(ctx, s.updateB(opt, body), jsonmodels.NewTrustedCertificate)
}

func (s *Service) updateB(opt UpdateOptions, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(paramFingerprint, opt.Fingerprint).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiTrustedCertificate)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// DeleteOptions options to delete a tenant
type DeleteOptions struct {
	TenantID string `url:"-"`

	Fingerprint string `url:"-"`
}

// Delete a trusted certificate
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(opt)).IgnoreNotFound()
}

func (s *Service) deleteB(opt DeleteOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(paramFingerprint, opt.Fingerprint).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiTrustedCertificate)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

/*
  Proof of Possession
*/
// ProofOptions proof of possession verification options
type ProofOptions struct {
	Fingerprint string `url:"-"`

	TenantID string `url:"-"`

	// Verification code. If left blank then it will be fetched
	Code string `url:"-"`

	// Path to the private key to use to verify the code
	PrivateKey string `url:"-"`
}

func (s *Service) ProofEndToEnd(ctx context.Context, opt ProofOptions) op.Result[jsonmodels.TrustedCertificate] {
	if opt.PrivateKey != "" {
		key, err := certutil.PrivateKeyFromFile(opt.PrivateKey)
		if err != nil {
			return op.Failed[jsonmodels.TrustedCertificate](fmt.Errorf("failed to get private key. %w", err), false)
		}
		signer, err := certutil.NewSignerFromKey(key)
		if err != nil {
			return op.Failed[jsonmodels.TrustedCertificate](fmt.Errorf("failed to create signer for private key. %w", err), false)
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
			return op.Failed[jsonmodels.TrustedCertificate](fmt.Errorf("failed to sign verification code. %w", err), false)
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
	return httpcore.Execute(ctx, s.proofB(opt, body), jsonmodels.NewTrustedCertificate)
}

func (s *Service) proofB(opt ProofOptions, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(paramFingerprint, opt.Fingerprint).
		SetBody(body).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiProofOfPossession)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

type CreateVerificationCodeOptions struct {
	Fingerprint string `url:"-"`

	TenantID string `url:"-"`
}

// Generate a verification code for the proof of possession operation for the certificate (by a given fingerprint)
func (s *Service) CreateVerificationCode(ctx context.Context, opt CreateVerificationCodeOptions) op.Result[jsonmodels.TrustedCertificate] {
	return httpcore.Execute(ctx, s.createVerificationCodeB(opt), jsonmodels.NewTrustedCertificate)
}

func (s *Service) createVerificationCodeB(opt CreateVerificationCodeOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(paramFingerprint, opt.Fingerprint).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiGenerateVerificationCode)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type ConfirmOptions struct {
	Fingerprint string `url:"-"`

	TenantID string `url:"-"`
}

// Confirm the proof of possession of an already uploaded certificate (by a given fingerprint) for a specific tenant
// TODO: This api calls always returns a 403 error
func (s *Service) Confirm(ctx context.Context, opt ConfirmOptions) op.Result[jsonmodels.TrustedCertificate] {
	return httpcore.Execute(ctx, s.confirmB(opt), jsonmodels.NewTrustedCertificate)
}

func (s *Service) confirmB(opt ConfirmOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(paramFingerprint, opt.Fingerprint).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiProofOfPossessionConfirm)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
