package nats

import (
	nats "github.com/nats-io/nats.go"
	"github.com/spf13/viper"
)

const (
	mailSubj = "nubes3_mail"
)

var (
	nc *nats.Conn
	c  *nats.EncodedConn
)

func InitNats() error {
	url := viper.GetString("NATS_URL")

	var err error
	nc, err = nats.Connect("nats://" + url)
	if err != nil {
		return err
	}
	c, err = nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return err
	}

	return nil
}

func CleanUp() {
	if nc != nil {
		nc.Close()
	}
}
