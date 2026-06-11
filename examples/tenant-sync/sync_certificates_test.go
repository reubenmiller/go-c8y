package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/certutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCertificatePEM generates a self-signed certificate and returns its PEM
// encoding together with the parsed certificate
func newCertificatePEM(t *testing.T, commonName string) ([]byte, *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	cert, err := certutil.NewSelfSignedCertificate(key, commonName)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: certutil.CertificateBlockType, Bytes: cert.Raw}), cert
}

func writeFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, content, 0o644))
	return path
}

func TestLoadTrustedCertificates(t *testing.T) {
	t.Run("single PEM certificate", func(t *testing.T) {
		pemBytes, cert := newCertificatePEM(t, "device-ca")
		path := writeFile(t, t.TempDir(), "device-ca.pem", pemBytes)

		certs, err := loadTrustedCertificates(path, "")
		require.NoError(t, err)
		require.Len(t, certs, 1)
		assert.Equal(t, "device-ca", certs[0].Name)
		sum := sha1.Sum(cert.Raw)
		assert.Equal(t, hex.EncodeToString(sum[:]), certs[0].Fingerprint)
		assert.Equal(t, base64.StdEncoding.EncodeToString(cert.Raw), certs[0].CertInPemFormat)
	})

	t.Run("manifest name overrides the common name", func(t *testing.T) {
		pemBytes, _ := newCertificatePEM(t, "device-ca")
		path := writeFile(t, t.TempDir(), "device-ca.pem", pemBytes)

		certs, err := loadTrustedCertificates(path, "renamed")
		require.NoError(t, err)
		require.Len(t, certs, 1)
		assert.Equal(t, "renamed", certs[0].Name)
	})

	t.Run("name falls back to the filename without extension", func(t *testing.T) {
		pemBytes, _ := newCertificatePEM(t, "")
		path := writeFile(t, t.TempDir(), "factory.pem", pemBytes)

		certs, err := loadTrustedCertificates(path, "")
		require.NoError(t, err)
		require.Len(t, certs, 1)
		assert.Equal(t, "factory", certs[0].Name)
	})

	t.Run("chain file yields every certificate", func(t *testing.T) {
		rootPEM, _ := newCertificatePEM(t, "root-ca")
		intermediatePEM, _ := newCertificatePEM(t, "intermediate-ca")
		path := writeFile(t, t.TempDir(), "chain.pem", append(rootPEM, intermediatePEM...))

		certs, err := loadTrustedCertificates(path, "")
		require.NoError(t, err)
		require.Len(t, certs, 2)
		assert.Equal(t, "root-ca", certs[0].Name)
		assert.Equal(t, "intermediate-ca", certs[1].Name)
	})

	t.Run("name with multiple certificates is rejected", func(t *testing.T) {
		rootPEM, _ := newCertificatePEM(t, "root-ca")
		intermediatePEM, _ := newCertificatePEM(t, "intermediate-ca")
		path := writeFile(t, t.TempDir(), "chain.pem", append(rootPEM, intermediatePEM...))

		_, err := loadTrustedCertificates(path, "single")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "'name' is set")
	})

	t.Run("non-certificate PEM blocks are skipped", func(t *testing.T) {
		keyPEM, err := certutil.MakeEllipticPrivateKeyPEM()
		require.NoError(t, err)
		certPEM, _ := newCertificatePEM(t, "device-ca")
		path := writeFile(t, t.TempDir(), "bundle.pem", append(keyPEM, certPEM...))

		certs, err := loadTrustedCertificates(path, "")
		require.NoError(t, err)
		require.Len(t, certs, 1)
		assert.Equal(t, "device-ca", certs[0].Name)
	})

	t.Run("DER certificate", func(t *testing.T) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		cert, err := certutil.NewSelfSignedCertificate(key, "der-ca")
		require.NoError(t, err)
		path := writeFile(t, t.TempDir(), "der-ca.cer", cert.Raw)

		certs, err := loadTrustedCertificates(path, "")
		require.NoError(t, err)
		require.Len(t, certs, 1)
		assert.Equal(t, "der-ca", certs[0].Name)
	})

	t.Run("non-certificate file is rejected", func(t *testing.T) {
		path := writeFile(t, t.TempDir(), "not-a-cert.pem", []byte("hello"))
		_, err := loadTrustedCertificates(path, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no certificates found")
	})
}

func TestLoadManifestCertificates(t *testing.T) {
	path := writeManifest(t, `
trustedCertificates:
  - autoRegistrationEnabled: true
    source:
      path: ./certificates
  - name: legacy-factory-ca
    status: DISABLED
    source:
      path: ./certificates/archive/legacy-factory-ca.pem

certificateRevocationLists:
  - source:
      path: ./certificates/revoked.csv
`)
	manifest, err := LoadManifest(path)
	require.NoError(t, err)

	require.Len(t, manifest.TrustedCertificates, 2)
	require.NotNil(t, manifest.TrustedCertificates[0].AutoRegistrationEnabled)
	assert.True(t, *manifest.TrustedCertificates[0].AutoRegistrationEnabled)
	assert.Nil(t, manifest.TrustedCertificates[1].AutoRegistrationEnabled)
	assert.Equal(t, "DISABLED", manifest.TrustedCertificates[1].Status)
	require.Len(t, manifest.CertificateRevocationLists, 1)
}

func TestTrustedCertificateResolvedSourceDefaults(t *testing.T) {
	spec := TrustedCertificateSpec{Source: Source{Path: "./certificates"}}
	assert.Equal(t, []string{"*.pem", "*.crt", "*.cer"}, spec.resolvedSource().Patterns)

	// Explicit patterns win
	spec = TrustedCertificateSpec{Source: Source{Path: "./certificates", Patterns: []string{"root-*.pem"}}}
	assert.Equal(t, []string{"root-*.pem"}, spec.resolvedSource().Patterns)
}

func TestSyncTrustedCertificatesDryRun(t *testing.T) {
	syncer, dir := newTestSyncer(t)
	syncer.DryRun = true
	pemBytes, _ := newCertificatePEM(t, "device-ca")
	writeFile(t, dir, "device-ca.pem", pemBytes)
	writeFile(t, dir, "notes.txt", []byte("ignored"))

	err := syncer.SyncTrustedCertificates(context.Background(), []TrustedCertificateSpec{
		{Source: Source{Path: "."}},
	})
	require.NoError(t, err)

	// Only the certificate file patterns are picked up
	require.Len(t, syncer.Results, 1)
	assert.Equal(t, ActionPlanned, syncer.Results[0].Action)
	assert.Equal(t, SectionTrustedCertificates, syncer.Results[0].Section)
	assert.Equal(t, "device-ca.pem", syncer.Results[0].Item)
}

func TestSyncTrustedCertificatesNameRequiresSingleFile(t *testing.T) {
	syncer, dir := newTestSyncer(t)
	pemA, _ := newCertificatePEM(t, "a")
	pemB, _ := newCertificatePEM(t, "b")
	writeFile(t, dir, "a.pem", pemA)
	writeFile(t, dir, "b.pem", pemB)

	err := syncer.SyncTrustedCertificates(context.Background(), []TrustedCertificateSpec{
		{Name: "single", Source: Source{Path: "."}},
	})
	require.NoError(t, err)

	require.Len(t, syncer.Results, 1)
	assert.Equal(t, ActionFailed, syncer.Results[0].Action)
	assert.Contains(t, syncer.Results[0].Err.Error(), "'name' is set")
}

func TestCountRevocationEntries(t *testing.T) {
	write := func(t *testing.T, content string) string {
		t.Helper()
		return writeFile(t, t.TempDir(), "revoked.csv", []byte(content))
	}

	t.Run("with header", func(t *testing.T) {
		count, err := countRevocationEntries(write(t, "SERIALNO,DATE\n0123abc,2026-01-01T00:00:00Z\nFF00aa,\n"))
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("without header", func(t *testing.T) {
		count, err := countRevocationEntries(write(t, "0123abc\nff00aa\n"))
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("invalid serial is rejected", func(t *testing.T) {
		_, err := countRevocationEntries(write(t, "SERIALNO,DATE\nnot-hex,2026-01-01T00:00:00Z\n"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a certificate serial number")
	})

	t.Run("too many columns is rejected", func(t *testing.T) {
		_, err := countRevocationEntries(write(t, "0123abc,2026-01-01T00:00:00Z,extra\n"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "columns")
	})

	t.Run("empty file is rejected", func(t *testing.T) {
		_, err := countRevocationEntries(write(t, "SERIALNO,DATE\n"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no revocation entries")
	})
}

func TestCertificateRevocationListResolvedSourceDefaults(t *testing.T) {
	spec := CertificateRevocationListSpec{Source: Source{Path: "./crl"}}
	assert.Equal(t, []string{"*.csv"}, spec.resolvedSource().Patterns)
}

func TestSyncCertificateRevocationListsDryRun(t *testing.T) {
	syncer, dir := newTestSyncer(t)
	syncer.DryRun = true
	writeFile(t, dir, "revoked.csv", []byte("SERIALNO,DATE\n0123abc,\n"))
	writeFile(t, dir, "readme.md", []byte("ignored"))

	err := syncer.SyncCertificateRevocationLists(context.Background(), []CertificateRevocationListSpec{
		{Source: Source{Path: "."}},
	})
	require.NoError(t, err)

	require.Len(t, syncer.Results, 1)
	assert.Equal(t, ActionPlanned, syncer.Results[0].Action)
	assert.Equal(t, SectionCertificateRevocations, syncer.Results[0].Section)
	assert.Equal(t, "revoked.csv", syncer.Results[0].Item)
}
