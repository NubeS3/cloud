package nats

import (
	"encoding/json"
	"time"
)

type KeyPairLogMessage struct {
	Event
	Public       string `json:"public"`
	BucketId     string `json:"bucket_id"`
	GeneratorUid string `json:"generator_uid"`
}

func SendKeyPairEvent(pub, bid, uid, t string) error {
	jsonData, err := json.Marshal(KeyPairLogMessage{
		Event: Event{
			Type: t,
			Date: time.Now(),
		},
		Public:       pub,
		BucketId:     bid,
		GeneratorUid: uid,
	})

	if err != nil {
		return err
	}

	return sc.Publish(keyPairSubj, jsonData)
}

func GetKeyPairLog(limit, offset int) ([]KeyPairLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "All",
		Data:   nil,
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(keyPairSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []KeyPairLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func GetKeyPairLogByDate(from, to time.Time, limit, offset int) ([]KeyPairLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339)},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(keyPairSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []KeyPairLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func GetKeyPairLogByType(t string, limit, offset int) ([]KeyPairLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Type",
		Data:   []string{t},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(keyPairSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []KeyPairLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}
