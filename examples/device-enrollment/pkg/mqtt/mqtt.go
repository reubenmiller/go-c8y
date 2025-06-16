package mqtt

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

func stripHost(host string) string {
	if _, after, found := strings.Cut(host, "://"); found {
		return after
	}
	return host
}

type MqttClientOptions struct {
	Host     string
	Port     int
	ClientID string
	Username string
	KeyFile  string
	CertFile string
	CAPath   string
	Topic    string
	Payload  []byte
	Duration time.Duration

	ShowMosquittoCommand bool
}

func subscribeToBroker(opts MqttClientOptions) error {
	opts.Host = stripHost(opts.Host)
	clientCert, err := tls.LoadX509KeyPair(opts.CertFile, opts.KeyFile)
	if err != nil {
		return err
	}

	conn, err := autopaho.NewConnection(context.Background(), autopaho.ClientConfig{
		ServerUrls: []*url.URL{
			{
				Scheme: "ssl",
				Host:   fmt.Sprintf("%s:%d", opts.Host, opts.Port),
			},
		},
		CleanStartOnInitialConnection: true,
		ConnectUsername:               opts.Username,
		KeepAlive:                     20,
		SessionExpiryInterval:         60,
		TlsCfg: &tls.Config{
			Certificates: []tls.Certificate{clientCert},

			InsecureSkipVerify: false,
		},
		ClientConfig: paho.ClientConfig{
			ClientID: opts.ClientID,

			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pub paho.PublishReceived) (bool, error) {
					slog.Info("✉️  Received message.", "topic", pub.Packet.Topic, "payload_len", len(pub.Packet.Payload))
					fmt.Fprintf(os.Stdout, "%s\ntopic: %s (m%d, q%d)\n%s\n%s\n", strings.Repeat("-", 50), pub.Packet.Topic, pub.Packet.PacketID, pub.Packet.QoS, pub.Packet.Payload, strings.Repeat("-", 50))
					return false, nil
				},
			},
			OnServerDisconnect: func(d *paho.Disconnect) {
				slog.Info("📵 Disconnected from broker.", "reason", d.ReasonCode)
			},
		},
		OnConnectionUp: func(connman *autopaho.ConnectionManager, connack *paho.Connack) {
			slog.Info("✅ Successfully connected to the MQTT broker.", "CONNACK", connack.ReasonCode)
		},
		OnConnectError: func(err error) {
			slog.Error("🧨 MQTT connection error.", "error", err)
		},
	})
	if err != nil {
		return err
	}

	if opts.ShowMosquittoCommand {
		fmt.Fprintf(os.Stderr, "\n\n%s\n\n", strings.Join(BuildMosquittoSubArgs(opts), " "))
	}

	if err := conn.AwaitConnection(context.Background()); err != nil {
		return nil
	}

	isPublishing := len(opts.Payload) > 0

	if !isPublishing {
		sub, err := conn.Subscribe(context.Background(), &paho.Subscribe{
			Subscriptions: []paho.SubscribeOptions{
				{
					Topic: opts.Topic,
					QoS:   1,
				},
			},
		})
		if err != nil {
			return err
		}
		slog.Info("💪 Subscribed to topics.", "topic", opts.Topic, "pkid", sub.Packet().PacketID)
	} else {
		slog.Info("Publishing message.")
		pub, err := conn.Publish(context.Background(), &paho.Publish{
			Topic:   opts.Topic,
			Payload: opts.Payload,
			QoS:     1,
		})
		if err != nil {
			return err
		}
		slog.Info("Published message.", "topic", opts.Topic, "reason", pub.ReasonCode)
		conn.Disconnect(context.Background())
	}

	timeoutChannel := make(<-chan time.Time)
	if opts.Duration > 0 {
		timeoutChannel = time.After(opts.Duration)
	}

	// Wait for connect to be closed
	select {
	case <-timeoutChannel:
		os.Exit(0)
	case <-conn.Done():
		os.Exit(0)
	}
	return nil
}
