package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/trustedcertificates"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/pkg/certutil"
)

// resolvedSource returns the spec source with the default certificate file
// patterns applied when scanning a directory
func (spec TrustedCertificateSpec) resolvedSource() Source {
	source := spec.Source
	if source.Path != "" && len(source.Patterns) == 0 {
		source.Patterns = []string{"*.pem", "*.crt", "*.cer"}
	}
	return source
}

// loadTrustedCertificates reads the certificates of a file: every CERTIFICATE
// block of a PEM file (other blocks, e.g. private keys, are ignored), or a
// single DER/base64 encoded certificate. The desired name is the override
// from the manifest, falling back to the certificate subject common name and
// then the filename.
func loadTrustedCertificates(path, nameOverride string) ([]*model.TrustedCertificate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var certs []*model.TrustedCertificate
	rest := raw
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != certutil.CertificateBlockType {
			continue
		}
		cert, err := model.NewTrustedCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("invalid certificate: %w", err)
		}
		certs = append(certs, cert)
	}

	if len(certs) == 0 {
		// Not PEM: try the whole file as a DER or base64 encoded certificate
		cert, err := model.NewTrustedCertificate(bytes.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("no certificates found (expected PEM, DER or base64 encoded DER): %w", err)
		}
		certs = append(certs, cert)
	}

	if nameOverride != "" && len(certs) > 1 {
		return nil, fmt.Errorf("file contains %d certificates but 'name' is set; name requires a single certificate", len(certs))
	}

	fallback := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	for _, cert := range certs {
		if nameOverride != "" {
			cert.Name = nameOverride
		} else if cert.Name == "" {
			cert.Name = fallback
		}
	}
	return certs, nil
}

// SyncTrustedCertificates uploads trusted device certificates into the tenant
// trust store. Certificates are matched by fingerprint: missing ones are
// created, and the name, status and autoRegistrationEnabled of existing ones
// are updated when they differ. Certificates are never deleted.
func (s *Syncer) SyncTrustedCertificates(ctx context.Context, specs []TrustedCertificateSpec) error {
	// Fetched lazily once the first certificate file is parsed
	var existing map[string]jsonmodels.TrustedCertificate

	for index, spec := range specs {
		label := fmt.Sprintf("trustedCertificates[%d]", index)
		files, ok := s.resolveSource(SectionTrustedCertificates, label, spec.resolvedSource())
		if !ok {
			continue
		}

		if spec.Name != "" && len(files) > 1 {
			s.record(SectionTrustedCertificates, spec.Name, ActionFailed, "",
				fmt.Errorf("source resolved to %d files but 'name' is set; use patterns to select a single file", len(files)))
			continue
		}

		for _, file := range files {
			if s.DryRun {
				item := spec.Name
				if item == "" {
					item = file.Filename
				}
				s.record(SectionTrustedCertificates, item, ActionPlanned, "ensure trusted certificate from "+file.Filename, nil)
				continue
			}

			if file.Path == "" {
				s.record(SectionTrustedCertificates, file.Filename, ActionFailed, "",
					fmt.Errorf("certificate source must provide a local file (url/linkOnly sources are not supported)"))
				continue
			}

			certs, err := loadTrustedCertificates(file.Path, spec.Name)
			if err != nil {
				s.record(SectionTrustedCertificates, file.Filename, ActionFailed, "parse certificate", err)
				continue
			}

			if existing == nil {
				existing, err = s.listTrustedCertificates(ctx)
				if err != nil {
					return fmt.Errorf("failed to list trusted certificates: %w", err)
				}
			}

			for _, cert := range certs {
				s.syncTrustedCertificate(ctx, spec, cert, file.Filename, existing)
			}
		}
	}
	return nil
}

// syncTrustedCertificate ensures a single certificate exists with the desired
// name/status/autoRegistrationEnabled, recording the outcome
func (s *Syncer) syncTrustedCertificate(ctx context.Context, spec TrustedCertificateSpec, cert *model.TrustedCertificate, filename string, existing map[string]jsonmodels.TrustedCertificate) {
	// Desired state: the status defaults to ENABLED and autoRegistration is
	// only managed when set in the manifest
	if spec.Status != "" {
		cert.Status = model.TrustedCertificateStatus(spec.Status)
	}
	cert.AutoRegistrationEnabled = spec.AutoRegistrationEnabled

	current, found := existing[strings.ToLower(cert.Fingerprint)]
	if !found {
		result := s.Client.TrustedCertificates.Create(ctx, trustedcertificates.CreateOptions{}, cert)
		action := ActionCreated
		if result.Status == op.StatusDuplicate {
			// 409: already trusted (e.g. by the platform or a parent tenant)
			action = ActionUnchanged
		}
		s.record(SectionTrustedCertificates, cert.Name, action, filename, result.Err)
		if result.Err == nil && result.Data.Fingerprint() != "" {
			existing[strings.ToLower(result.Data.Fingerprint())] = result.Data
		}
		return
	}

	upToDate := current.Name() == cert.Name &&
		current.Status() == string(cert.Status) &&
		(cert.AutoRegistrationEnabled == nil || current.Get("autoRegistrationEnabled").Bool() == *cert.AutoRegistrationEnabled)
	if upToDate {
		s.record(SectionTrustedCertificates, cert.Name, ActionUnchanged, filename, nil)
		return
	}

	body := map[string]any{
		"name":   cert.Name,
		"status": cert.Status,
	}
	if cert.AutoRegistrationEnabled != nil {
		body["autoRegistrationEnabled"] = *cert.AutoRegistrationEnabled
	}
	result := s.Client.TrustedCertificates.Update(ctx, trustedcertificates.UpdateOptions{
		Fingerprint: current.Fingerprint(),
	}, body)
	s.record(SectionTrustedCertificates, cert.Name, ActionUpdated, fmt.Sprintf("status=%s", cert.Status), result.Err)
}

// listTrustedCertificates fetches the trusted certificates of the target
// tenant, keyed by their lowercase fingerprint
func (s *Syncer) listTrustedCertificates(ctx context.Context) (map[string]jsonmodels.TrustedCertificate, error) {
	certs := make(map[string]jsonmodels.TrustedCertificate)
	result := s.Client.TrustedCertificates.List(ctx, trustedcertificates.ListOptions{
		PaginationOptions: pagination.PaginationOptions{PageSize: 2000},
	})
	for cert, err := range op.Iter2(result) {
		if err != nil {
			return nil, err
		}
		certs[strings.ToLower(cert.Fingerprint())] = cert
	}
	return certs, nil
}

// resolvedSource returns the spec source with the *.csv default pattern applied
func (spec CertificateRevocationListSpec) resolvedSource() Source {
	source := spec.Source
	if source.Path != "" && len(source.Patterns) == 0 {
		source.Patterns = []string{"*.csv"}
	}
	return source
}

// countRevocationEntries validates a revocation list CSV file (header
// SERIALNO[,DATE]; one serial number in hex per row) and returns the number
// of entries, catching malformed files before they are uploaded
func countRevocationEntries(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1

	count := 0
	for line := 1; ; line++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		if len(record) > 2 {
			return 0, fmt.Errorf("line %d: expected SERIALNO[,DATE] but got %d columns", line, len(record))
		}
		serial := strings.TrimSpace(record[0])
		if line == 1 && strings.EqualFold(serial, "SERIALNO") {
			continue // header row
		}
		if serial == "" || strings.Trim(strings.ToLower(serial), "0123456789abcdef") != "" {
			return 0, fmt.Errorf("line %d: %q is not a certificate serial number in hex", line, serial)
		}
		count++
	}
	if count == 0 {
		return 0, fmt.Errorf("no revocation entries found")
	}
	return count, nil
}

// SyncCertificateRevocationLists uploads certificate revocation entries from
// CSV files. Uploads are additive: entries already on the tenant revocation
// list are never removed.
func (s *Syncer) SyncCertificateRevocationLists(ctx context.Context, specs []CertificateRevocationListSpec) error {
	for index, spec := range specs {
		label := fmt.Sprintf("certificateRevocationLists[%d]", index)
		files, ok := s.resolveSource(SectionCertificateRevocations, label, spec.resolvedSource())
		if !ok {
			continue
		}

		for _, file := range files {
			if s.DryRun {
				s.record(SectionCertificateRevocations, file.Filename, ActionPlanned, "upload revocation entries", nil)
				continue
			}

			if file.Path == "" {
				s.record(SectionCertificateRevocations, file.Filename, ActionFailed, "",
					fmt.Errorf("revocation list source must provide a local file (url/linkOnly sources are not supported)"))
				continue
			}

			count, err := countRevocationEntries(file.Path)
			if err != nil {
				s.record(SectionCertificateRevocations, file.Filename, ActionFailed, "parse revocation list", err)
				continue
			}

			result := s.Client.TrustedCertificates.RevocationList.AddFile(ctx, core.UploadFileOptions{
				FilePath:    file.Path,
				Name:        file.Filename,
				ContentType: "text/csv",
			})
			s.record(SectionCertificateRevocations, file.Filename, ActionUpdated,
				fmt.Sprintf("%d revocation entrie(s)", count), result.Err)
		}
	}
	return nil
}
