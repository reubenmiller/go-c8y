package main

import (
	"os"
	"path/filepath"
	"testing"
)

const builderSrc = `package p

import "resty.dev/v3"

var ApiItems = "/inventory/managedObjects"
var ApiFeature = "/features/{key}"

func (s *Service) listB() {
	s.R().SetMethod(resty.MethodGet).SetURL(ApiItems)
}

func (s *Service) createB() {
	s.R().SetMethod(resty.MethodPost).SetURL(ApiItems)
}

func (s *Service) getFeatureB() {
	s.R().SetMethod(resty.MethodGet).SetURL(ApiFeature)
}
`

func writeSrc(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "api.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLintReportsMissingMethodOnExistingPath(t *testing.T) {
	doc := &OAS{Paths: map[string]PathItem{
		"/inventory/managedObjects": {Get: &Operation{}, Post: &Operation{}, Put: &Operation{}},
		"/features/{featureKey}":    {Get: &Operation{}},
	}}
	res, err := Lint(doc, writeSrc(t, builderSrc), driftWaivers{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.MissingInSDK) != 1 || res.MissingInSDK[0] != "PUT /inventory/managedObjects" {
		t.Fatalf("missing = %v, want only PUT /inventory/managedObjects", res.MissingInSDK)
	}
}

func TestLintMethodSpecificWaiver(t *testing.T) {
	doc := &OAS{Paths: map[string]PathItem{
		"/tenant/trusted-certificates/settings/crl": {Get: &Operation{}, Post: &Operation{}},
	}}
	res, err := Lint(doc, writeSrc(t, "package p\n"), driftWaivers{
		IgnoreMissing: []string{"POST /tenant/trusted-certificates/settings/crl"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.MissingInSDK) != 1 || res.MissingInSDK[0] != "GET /tenant/trusted-certificates/settings/crl" {
		t.Fatalf("missing = %v, want only the unwaived GET", res.MissingInSDK)
	}
	if res.WaivedCount != 1 {
		t.Errorf("waived = %d, want 1", res.WaivedCount)
	}
}

func TestLintUnknownMethodCoversAllMethods(t *testing.T) {
	doc := &OAS{Paths: map[string]PathItem{"/alarm/alarms": {Get: &Operation{}, Post: &Operation{}}}}
	// A path literal with no SetMethod/SetURL pairing: methods are unknown.
	res, err := Lint(doc, writeSrc(t, "package p\nvar a = \"/alarm/alarms\"\n"), driftWaivers{})
	if err != nil {
		t.Fatal(err)
	}
	if res.HasDrift() {
		t.Fatalf("unexpected drift %+v", res)
	}
}

func TestCheckWaiverReasons(t *testing.T) {
	overlay := `resources: {}
drift:
  ignoreMissing:
    # Group comment gives these a reason.
    - /alarm
    - /event

    - /no/reason/here
    - /inline # has a reason
  ignoreExtra:
    - /also/no/reason
`
	path := filepath.Join(t.TempDir(), "overlay.yml")
	if err := os.WriteFile(path, []byte(overlay), 0o644); err != nil {
		t.Fatal(err)
	}
	missing, err := CheckWaiverReasons(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 2 {
		t.Fatalf("missing reasons = %v, want 2 entries", missing)
	}
}
