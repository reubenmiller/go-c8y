package mqtt

import (
	"fmt"
	"os"

	"github.com/reubenmiller/example/pkg/cli"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
)

type PublishCmd struct {
	Topic   string `name:"topic" short:"t" required:"" help:"MQTT Topic" type:"string"`
	Payload string `name:"payload" short:"m" required:"" help:"MQTT Payload to publish to the topic" type:"string"`
}

func (r *PublishCmd) Run(ctx *cli.Context) error {
	certPEM, err := os.ReadFile(ctx.CertFile)
	if err != nil {
		panic(fmt.Errorf("failed to read certificate file. %w", err))
	}

	cert, err := certutil.ParseCertificatePEM(certPEM)
	if err != nil {
		panic(fmt.Errorf("failed to parse certificate. %w", err))
	}
	fmt.Fprintf(os.Stderr, "\n📣 Using existing device certificate: externalID=%s\n", cert.Subject.CommonName)

	if err := subscribeToBroker(MqttClientOptions{
		Host:     ctx.Host,
		Port:     9883,
		Topic:    r.Topic,
		ClientID: cert.Subject.CommonName,
		Username: cert.Issuer.CommonName,
		KeyFile:  ctx.KeyFile,
		CertFile: ctx.CertFile,
		CAPath:   ctx.CAPath,
		Payload:  []byte(r.Payload),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "🚫 Failed to publish to broker. %s\n", err)
		os.Exit(1)
	}

	return nil
}
