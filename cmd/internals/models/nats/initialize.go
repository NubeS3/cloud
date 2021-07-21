package nats

import (
	"github.com/nats-io/nats.go"
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
	contextExpTime = time.Second * 30
)

var (
	//sc stan.Conn
	nc *nats.Conn
	js nats.JetStreamContext
)

func InitNats() error {
	url := viper.GetString("NATS_URL")
	//clusterId := viper.GetString("STAN_CLUSTER_ID")

	println("connecting to nats at: " + url)
	var err error
	//sc, err = stan.Connect(clusterId, "nubes3"+randstr.GetString(8), stan.NatsURL("nats://"+url))
	//if err != nil {
	//	return err
	//}

	nc, err = nats.Connect("nats://" + url)
	if err != nil {
		return err
	}

	js, err = nc.JetStream()
	if err != nil {
		return err
	}

	info, err := js.AddStream(&nats.StreamConfig{
		Name:     "NUBES3",
		Subjects: []string{"NUBES3.*"},
	})
	if err != nil {
		println(err)
	} else {
		println(info.Config.Name + " stream created at " + info.Created.String())
	}

	return nil
}

func CleanUp() {
	//if sc != nil {
	//	_ = sc.Close()
	//}
	if nc != nil {
		nc.Close()
	}
}
