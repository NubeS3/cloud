package nats

import (
	stan "github.com/nats-io/stan.go"
	"github.com/spf13/viper"
)

const (
	mailSubj       = "nubes3_mail"
	errSubj        = "nubes3_err"
	uploadFileSubj = "nubes3_upload_file"
)

var (
	sc stan.Conn
)

func InitNats() error {
	url := viper.GetString("NATS_URL")

	var err error
	sc, err = stan.Connect("log", "log-1", stan.NatsURL("nats://"+url))
	if err != nil {
		return err
	}

	return nil
}

func CleanUp() {
	if sc != nil {
		_ = sc.Close()
	}
}
