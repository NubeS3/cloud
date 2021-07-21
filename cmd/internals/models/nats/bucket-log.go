package nats

import (
	"encoding/json"
	"time"
)

type BucketLogMessage struct {
	Event
	Id     string `json:"id"`
	Uid    string `json:"uid"`
	Name   string `json:"name"`
	Region string `json:"region"`
}

func SendBucketEvent(id, uid, name, region, t string) error {
	jsonData, err := json.Marshal(BucketLogMessage{
		Event: Event{
			Type: t,
			Date: time.Now(),
		},
		Id:     id,
		Uid:    uid,
		Name:   name,
		Region: region,
	})

	if err != nil {
		return err
	}

	//return sc.Publish(bucketSubj, jsonData)
	_, err = js.Publish("NUBES3."+bucketSubj, jsonData)
	return err
}

func GetBucketLog(limit, offset int) ([]BucketLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "All",
		Data:   nil,
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(bucketSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []BucketLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func GetBucketLogByDate(from, to time.Time, limit, offset int) ([]BucketLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339)},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(bucketSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []BucketLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func GetBucketLogByType(t string, limit, offset int) ([]BucketLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Type",
		Data:   []string{t},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(bucketSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []BucketLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}
