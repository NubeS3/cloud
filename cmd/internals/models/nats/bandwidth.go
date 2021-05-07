package nats

import (
	"encoding/json"
	"time"
)

type BandwidthLog struct {
	Event
	Size int64  `json:"size"`
	Uid  string `json:"uid"`
	From string `json:"from"`
}

func SendBandwidthLog(size int64, uid, from, sourceType string) error {
	jsonData, err := json.Marshal(BandwidthLog{
		Event: Event{
			Type: sourceType,
			Date: time.Now(),
		},
		Size: size,
		Uid:  uid,
		From: from,
	})

	if err != nil {
		return err
	}

	return sc.Publish(bandwidthSubj, jsonData)
}
