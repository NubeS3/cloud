package nats

import (
	"github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	"github.com/spf13/viper"
	"time"
)

const (
	mailSubj       = "nubes3_mail"
	errSubj        = "nubes3_err"
	fileSubject    = "nubes3_file"
	reqSubj        = "nubes3_req_"
	bandwidthSubj  = "nubes3_bandwidth"
	userSubj       = "nubes3_user"
	bucketSubj     = "nubes3_bucket"
	folderSubj     = "nubes3_folder"
	accessKeySubj  = "nubes3_accessKey"
	keyPairSubj    = "nubes3_keyPair"
	contextExpTime = time.Second * 10
)

var (
	sc stan.Conn
	nc *nats.Conn
)

func InitNats() error {
	url := viper.GetString("NATS_URL")
	clusterId := viper.GetString("STAN_CLUSTER_ID")

	var err error
	sc, err = stan.Connect(clusterId, "nubes3", stan.NatsURL("nats://"+url))
	if err != nil {
		return err
	}

	nc, err = nats.Connect("nats://" + url)
	if err != nil {
		return err
	}

	return nil
}

func CleanUp() {
	if sc != nil {
		_ = sc.Close()
	}
	if nc != nil {
		nc.Close()
	}
}
