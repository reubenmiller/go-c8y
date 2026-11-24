// Package dataprep provides access to the Cumulocity Data Preparation API
// (/service/dataprep): deploying and removing rules (smart functions),
// running smart function tests, and downloading diagnostics.
//
// Rules are deployed as packages (a tar.gz with data-prep.yaml, the
// bundled smart function and tests). RulePackage builds and checks them in
// memory, so callers never handle the archive format directly.
//
// The rule read endpoints used by List, Get, GetDraft, Exists, Apply and
// Export (GET /v1/rules, /v1/rules/{name}/deployed and /draft) are used by
// the Data Preparation application but are not in the published OpenAPI
// specification, so they may change.
package dataprep

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
)

// API paths
const (
	ApiRules        = "/service/dataprep/v1/rules"
	ApiRule         = "/service/dataprep/v1/rules/{ruleName}"
	ApiRuleDeployed = "/service/dataprep/v1/rules/{ruleName}/deployed"
	ApiRuleDraft    = "/service/dataprep/v1/rules/{ruleName}/draft"
	ApiRunTests     = "/service/dataprep/v1/run-tests"
	ApiDiagnostics  = "/service/dataprep/v1/diagnostics"
)

// ParamRuleName is the rule name path parameter.
const ParamRuleName = "ruleName"

// Content types
const (
	MimeTypeGzip = "application/gzip"
	MimeTypeZip  = "application/zip"
)

// Service is the entry point to the Data Preparation API.
type Service struct {
	Rules       *RuleService
	Tests       *TestService
	Diagnostics *DiagnosticsService
}

// NewService creates the Data Preparation service.
func NewService(s *httpcore.Service) *Service {
	tests := &TestService{svc: *s}
	return &Service{
		Rules:       &RuleService{svc: *s, tests: tests},
		Tests:       tests,
		Diagnostics: &DiagnosticsService{svc: *s},
	}
}
