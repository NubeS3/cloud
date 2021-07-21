package nats

import (
	"encoding/json"
	"strconv"
	"time"
)

type BandwidthLog struct {
	Event
	Size     int64  `json:"size"`
	BucketId string `json:"bucket_id"`
	Uid      string `json:"uid"`
	From     string `json:"from"`
}

func SendBandwidthLog(size int64, uid, bucketId, from, sourceType string) error {
	jsonData, err := json.Marshal(BandwidthLog{
		Event: Event{
			Type: sourceType,
			Date: time.Now(),
		},
		BucketId: bucketId,
		Size:     size,
		Uid:      uid,
		From:     from,
	})

	if err != nil {
		return err
	}

	//return sc.Publish(bandwidthSubj, jsonData)
	_, err = js.Publish("NUBES3."+bandwidthSubj, jsonData)
	return err
}

func SumBandwidthByDateRangeWithUid(uid string, from, to time.Time) (float64, error) {
	request := Req{
		Limit:  0,
		Offset: 0,
		Type:   "Uid",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339), uid},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(bandwidthSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return 0, err
	}

	total, _ := strconv.ParseFloat(string(res.Data), 64)
	return total, nil
}

func SumBandwidthByDateRangeWithBucketId(bid string, from, to time.Time) (float64, error) {
	request := Req{
		Limit:  0,
		Offset: 0,
		Type:   "Bid",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339), bid},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(bandwidthSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return 0, err
	}

	total, _ := strconv.ParseFloat(string(res.Data), 64)
	return total, nil
}

func SumBandwidthByDateRangeWithFrom(fromSrc string, from, to time.Time) (float64, error) {
	request := Req{
		Limit:  0,
		Offset: 0,
		Type:   "From",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339), fromSrc},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(bandwidthSubj+"query", jsonReq, contextExpTime)
	if err != nil {
		return 0, err
	}

	total, _ := strconv.ParseFloat(string(res.Data), 64)
	return total, nil
}
