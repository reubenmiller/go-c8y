package certutil

import (
	"bytes"
	"crypto/tls"
	"encoding/pem"
)

type CertificateChain []tls.Certificate

func (o CertificateChain) Header() ([]byte, error) {
	pemCerts := bytes.Buffer{}
	for _, cert := range o {
		for i := 1; i < len(cert.Certificate); i++ {
			if _, writeErr := pemCerts.Write(bytes.ReplaceAll(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[i]}), []byte("\n"), []byte(" "))); writeErr != nil {
				return nil, writeErr
			}
		}
	}
	return pemCerts.Bytes(), nil
}
