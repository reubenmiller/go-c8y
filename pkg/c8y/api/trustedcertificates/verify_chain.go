package trustedcertificates

import (
	"bytes"
	"context"
	"encoding/pem"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// apiVerifyCertChain verifies a certificate chain against a tenant.
var apiVerifyCertChain = "/tenant/trusted-certificates/verify-cert-chain"

// HeaderClientCertChain carries a certificate chain in a header.
const HeaderClientCertChain = "X-Cumulocity-Client-Cert-Chain"

// VerifyChainOptions are the inputs of VerifyChain.
type VerifyChainOptions struct {
	// Chain is the leaf certificate, or the chain, in PEM format (up to 10 certificates).
	Chain []byte

	// Tenant is the tenant to verify against. When empty, the platform resolves
	// it from the chain.
	Tenant string
}

// VerifyChain checks a certificate chain against the tenant's trusted
// certificates, including revocation when CRL checks are enabled. It needs
// tenant management read or admin rights, or service user credentials.
func (s *Service) VerifyChain(ctx context.Context, opt VerifyChainOptions) op.Result[jsonmodels.VerifyCertificateChain] {
	return httpcore.Execute(ctx, s.verifyChainB(opt), jsonmodels.NewVerifyCertificateChain)
}

func (s *Service) verifyChainB(opt VerifyChainOptions) *httpcore.TryRequest {
	form := map[string]string{}
	if opt.Tenant != "" {
		form["tenantId"] = opt.Tenant
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader(HeaderClientCertChain, ChainHeader(opt.Chain)).
		SetFileReader("file", "chain.pem", bytes.NewReader(opt.Chain)).
		SetMultipartFormData(form).
		SetURL(apiVerifyCertChain)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ChainHeader formats a PEM chain for the X-Cumulocity-Client-Cert-Chain
// header: certificates separated by "," and line breaks replaced by spaces.
func ChainHeader(chain []byte) string {
	var parts []string
	rest := chain
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		encoded := strings.TrimSpace(string(pem.EncodeToMemory(block)))
		parts = append(parts, strings.ReplaceAll(encoded, "\n", " "))
	}
	return strings.Join(parts, ",")
}
