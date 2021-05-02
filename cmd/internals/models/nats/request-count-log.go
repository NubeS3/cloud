package nats

import (
	"encoding/json"
	"strconv"
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
	var subjectSuffix string
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
		subjectSuffix = "unauth"
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
		subjectSuffix = "auth"
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
		subjectSuffix = "access-key"
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
		subjectSuffix = "signed"
	}

	if err != nil {
		return err
	}

	return sc.Publish(reqSubj+subjectSuffix, jsonData)
}

func ReadUnauthReqCount(limit, offset int) ([]UnauthReqLog, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "All",
		Data:   []string{},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"unauth"+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []UnauthReqLog
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func ReadUnauthReqCountByDateRange(from, to time.Time, limit, offset int) ([]UnauthReqLog, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339)},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"unauth"+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []UnauthReqLog
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func CountUnauthReqCountByDateRange(from, to time.Time, limit, offset int) (int64, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date-count",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339)},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"unauth"+"query", jsonReq, contextExpTime)
	if err != nil {
		return 0, err
	}

	count, _ := strconv.ParseInt(string(res.Data), 10, 64)
	return count, nil
}

//

func ReadAuthReqCountByDateRange(uid string, from, to time.Time, limit, offset int) ([]AuthReqLog, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339), uid},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"auth"+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []AuthReqLog
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func CountAuthReqCountByDateRange(uid string, from, to time.Time, limit, offset int) (int64, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339), uid},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"auth"+"query", jsonReq, contextExpTime)
	if err != nil {
		return 0, err
	}

	count, _ := strconv.ParseInt(string(res.Data), 10, 64)
	return count, nil
}

//

func ReadAccessKeyReqCountByDateRange(key string, from, to time.Time, limit, offset int) ([]AccessKeyReqLog, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339), key},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"access-key"+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []AccessKeyReqLog
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func CountAccessKeyReqCountByDateRange(key string, from, to time.Time, limit, offset int) (int64, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339), key},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"access-key"+"query", jsonReq, contextExpTime)
	if err != nil {
		return 0, err
	}

	count, _ := strconv.ParseInt(string(res.Data), 10, 64)
	return count, nil
}

func ReadAccessKeyReqCount(key string, limit, offset int) ([]AccessKeyReqLog, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "All",
		Data:   []string{key},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"access-key"+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []AccessKeyReqLog
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func CountAccessKeyReqCount(key string, limit, offset int) (int64, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "All",
		Data:   []string{key},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"access-key"+"query", jsonReq, contextExpTime)
	if err != nil {
		return 0, err
	}

	count, _ := strconv.ParseInt(string(res.Data), 10, 64)
	return count, nil
}

//

func ReadSignedReqCountByDateRange(public string, from, to time.Time, limit, offset int) ([]SignedReqLog, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339), public},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"signed"+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []SignedReqLog
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func CountSignedReqCountByDateRange(public string, from, to time.Time, limit, offset int) (int64, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339), public},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"signed"+"query", jsonReq, contextExpTime)
	if err != nil {
		return 0, err
	}

	count, _ := strconv.ParseInt(string(res.Data), 10, 64)
	return count, nil
}

func ReadSignedReqCount(public string, limit, offset int) ([]SignedReqLog, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{public},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"signed"+"query", jsonReq, contextExpTime)
	if err != nil {
		return nil, err
	}

	var logs []SignedReqLog
	_ = json.Unmarshal(res.Data, &logs)
	return logs, err
}

func CountSignedReqCount(public string, limit, offset int) (int64, error) {
	request := Req{
		Limit:  limit,
		Offset: offset,
		Type:   "Date",
		Data:   []string{public},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(reqSubj+"signed"+"query", jsonReq, contextExpTime)
	if err != nil {
		return 0, err
	}

	count, _ := strconv.ParseInt(string(res.Data), 10, 64)
	return count, nil
}
