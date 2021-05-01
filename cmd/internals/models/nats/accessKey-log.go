package nats

import (
	"encoding/json"
	"time"
)

type AccessKeyLogMessage struct {
	Event
	Key      string `json:"id"`
	BucketId string `json:"bid"`
	Uid      string `json:"uid"`
	Content  string `json:"content"`
}

func SendAccessKeyEvent(key, bid, uid, content, t string) error {
	jsonData, err := json.Marshal(AccessKeyLogMessage{
		Event: Event{
			Type: t,
			Date: time.Now(),
		},
		Key:      key,
		BucketId: bid,
		Uid:      uid,
		Content:  content,
	})

	if err != nil {
		return err
	}

	return sc.Publish(accessKeySubj, jsonData)
}

func GetAccessKeyLog(limit, offset int) ([]AccessKeyLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "All",
		Data:   nil,
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(accessKeySubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []AccessKeyLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func GetAccessKeyLogByDate(from, to time.Time, limit, offset int) ([]AccessKeyLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339)},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(accessKeySubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []AccessKeyLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func GetAccessKeyLogByType(t string, limit, offset int) ([]AccessKeyLogMessage, error) {
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

	var logs []AccessKeyLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}
