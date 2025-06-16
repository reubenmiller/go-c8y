package mqtt

import (
	"fmt"
	"os"
	"time"

	"github.com/reubenmiller/example/pkg/cli"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
)

/*
Subscribe command
*/
type SubscribeCmd struct {
	Topic    string        `name:"topic" short:"t" required:"" help:"MQTT Topic" type:"string"`
	Duration time.Duration `name:"duration" short:"W" help:"Time to subscribe to a topic to"`
}

func (r *SubscribeCmd) Run(ctx *cli.Context) error {
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
		Port:     ctx.Port,
		Topic:    r.Topic,
		ClientID: cert.Subject.CommonName,
		Username: cert.Issuer.CommonName,
		KeyFile:  ctx.KeyFile,
		CertFile: ctx.CertFile,
		CAPath:   ctx.CAPath,
		Duration: r.Duration,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "🚫 Failed to subscribe to broker. %s\n", err)
		os.Exit(1)
	}

	return nil
}
