package nats

import (
	"encoding/json"
	"time"
)

type UserLogMessage struct {
	Event
	Uid      string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func SendUserEvent(uid, username, email, t string) error {
	jsonData, err := json.Marshal(UserLogMessage{
		Event: Event{
			Type: t,
			Date: time.Now(),
		},
		Uid:      uid,
		Username: username,
		Email:    email,
	})

	if err != nil {
		return err
	}

	return sc.Publish(bucketSubj, jsonData)
}

func GetUserLog(limit, offset int) ([]UserLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "All",
		Data:   nil,
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(userSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []UserLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func GetUserLogByDate(from, to time.Time, limit, offset int) ([]UserLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339)},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(userSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []UserLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func GetUserLogByType(t string, limit, offset int) ([]UserLogMessage, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Type",
		Data:   []string{t},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(userSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []UserLogMessage
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}
