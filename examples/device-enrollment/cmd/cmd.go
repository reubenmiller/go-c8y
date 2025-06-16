package cmd

import (
	"github.com/alecthomas/kong"
	"github.com/reubenmiller/example/pkg/cli"
	"github.com/reubenmiller/example/pkg/enroll"
	"github.com/reubenmiller/example/pkg/mqtt"
)

func NewContextFromCli(in *CLI) *cli.Context {
	return &cli.Context{
		Debug:    in.Debug,
		DeviceID: in.DeviceID,
		CertFile: in.Cert,
		KeyFile:  in.Key,
		CAPath:   in.CA,
		Host:     in.Host,
		Port:     in.Port,
	}
}

type CLI struct {
	Debug    bool   `help:"Enable debug mode."`
	DeviceID string `env:"DEVICE_ID" help:"Device external identity"`
	Key      string `default:"device.key" help:"Private key path. It will be created if it does not exist"`
	Cert     string `default:"device.crt" help:"Certificate path. It will be created if it does not exist"`
	CA       string `name:"ca" help:"CA file or path"`
	Host     string `name:"host" env:"C8Y_HOST,C8Y_URL,C8Y_BASEURL" help:"Cumulocity host." type:"string"`
	Port     int    `name:"port" default:"9883" help:"Cumulocity MQTT Port"`

	Subscribe mqtt.SubscribeCmd `cmd:"subscribe" help:"Subscribe to a topic"`
	Publish   mqtt.PublishCmd   `cmd:"publish" help:"Publish to a topic"`
	Enroll    enroll.EnrollCmd  `cmd:"enroll" help:"Enroll the device"`
}

func Run() error {
	cmdCtx := &CLI{}
	ctx := kong.Parse(cmdCtx)
	return ctx.Run(NewContextFromCli(cmdCtx))
}
