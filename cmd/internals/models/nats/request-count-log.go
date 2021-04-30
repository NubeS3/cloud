package nats

import (
	"encoding/json"
	"time"
)

type ReqLog struct {
	Method   string `json:"method"`
	Req      string `json:"req"`
	SourceIp string `json:"source_ip"`
}

type UnauthReqLog struct {
	Event
	ReqLog
}

type AuthReqLog struct {
	Event
	ReqLog
	UserId string `json:"user_id"`
}

type AccessKeyReqLog struct {
	Event
	ReqLog
	Key string `json:"key"`
}

type SignedReqLog struct {
	Event
	ReqLog
	Public string `json:"public"`
}

func SendReqCountEvent(data, t, method, source, req string) error {
	var jsonData []byte
	var err error
	if t == "Req" {
		jsonData, err = json.Marshal(UnauthReqLog{
			Event: Event{
				Type: t,
				Date: time.Now(),
			},
			ReqLog: ReqLog{
				Method:   method,
				SourceIp: source,
				Req:      req,
			},
		})
	} else if t == "Auth" {
		jsonData, err = json.Marshal(AuthReqLog{
			Event: Event{
				Type: t,
				Date: time.Now(),
			},
			ReqLog: ReqLog{
				Method:   method,
				SourceIp: source,
				Req:      req,
			},
			UserId: data,
		})
	} else if t == "AccessKey" {
		jsonData, err = json.Marshal(AccessKeyReqLog{
			Event: Event{
				Type: t,
				Date: time.Now(),
			},
			ReqLog: ReqLog{
				Method:   method,
				SourceIp: source,
				Req:      req,
			},
			Key: data,
		})
	} else if t == "Signed" {
		jsonData, err = json.Marshal(SignedReqLog{
			Event: Event{
				Type: t,
				Date: time.Now(),
			},
			ReqLog: ReqLog{
				Method:   method,
				SourceIp: source,
				Req:      req,
			},
			Public: data,
		})
	}

	if err != nil {
		return err
	}

	return sc.Publish(reqSubj, jsonData)
}
