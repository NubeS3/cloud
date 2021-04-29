package nats

import (
	"encoding/json"
	"time"
)

type ErrLogMessage struct {
	Event
	Content string `json:"content"`
}

func SendErrorEvent(content, t string) error {
	jsonData, err := json.Marshal(ErrLogMessage{
		Event: Event{
			Type: t,
			Date: time.Now(),
		},
		Content: content,
	})

	if err != nil {
		return err
	}

	return sc.Publish(errSubj, jsonData)
}

func GetErrLog(limit, offset int) ([]ErrLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "All",
		Data:   nil,
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(errSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []ErrLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func GetErrLogByDate(from, to time.Time, limit, offset int) ([]ErrLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339)},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(errSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []ErrLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func GetErrLogByType(t string, limit, offset int) ([]ErrLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{t},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(errSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []ErrLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}
