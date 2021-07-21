package nats

import (
	"encoding/json"
	"strconv"
	"time"
)

type Event struct {
	Type string    `json:"type"`
	Date time.Time `json:"at"`
}

type FileLog struct {
	Event
	Id         string    `json:"id"`
	FId        string    `json:"file_id"`
	FileName   string    `json:"file_name"`
	Size       int64     `json:"size"`
	BucketId   string    `json:"bucket_id"`
	UploadDate time.Time `json:"upload_date"`
	Uid        string    `json:"uid"`
}

func SendUploadFileEvent(id, fid, name string, size int64,
	bid string, uploadDate time.Time, uid string) error {
	jsonData, err := json.Marshal(FileLog{
		Event: Event{
			Type: "Upload",
			Date: time.Now(),
		},
		Id:         id,
		FId:        fid,
		FileName:   name,
		Size:       size,
		BucketId:   bid,
		UploadDate: uploadDate,
		Uid:        uid,
	})

	if err != nil {
		return err
	}

	//return sc.Publish(fileSubject, jsonData)
	println("NUBES3." + fileSubject)
	_, err = js.Publish("NUBES3."+fileSubject, jsonData)
	return err
}

func SendDownloadFileEvent(id, fid, name string, size int64,
	bid string, uploadDate time.Time, uid string) error {
	jsonData, err := json.Marshal(FileLog{
		Event: Event{
			Type: "Download",
			Date: time.Now(),
		},
		Id:         id,
		FId:        fid,
		FileName:   name,
		Size:       size,
		BucketId:   bid,
		UploadDate: uploadDate,
		Uid:        uid,
	})

	if err != nil {
		return err
	}

	//return sc.Publish(fileSubject, jsonData)
	_, err = js.Publish("NUBES3."+fileSubject, jsonData)
	return err
}

func SendDeleteFileEvent(id, fid, name string, size int64,
	bid string, deleteDate time.Time, uid string) error {
	jsonData, err := json.Marshal(FileLog{
		Event: Event{
			Type: "Delete",
			Date: time.Now(),
		},
		Id:         id,
		FId:        fid,
		FileName:   name,
		Size:       size,
		BucketId:   bid,
		UploadDate: deleteDate,
		Uid:        uid,
	})

	if err != nil {
		return err
	}

	//return sc.Publish(fileSubject, jsonData)
	_, err = js.Publish("NUBES3."+fileSubject, jsonData)
	return err
}

func GetAvgStoredSizeByUidInDateRange(uid string, from, to time.Time) (float64, error) {
	request := Req{
		Limit:  1000,
		Offset: 0,
		Type:   "AvgSize",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339), uid},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(fileSubject+"query", jsonReq, contextExpTime)
	if err != nil {
		return 0.0, err
	}

	avgSize, err := strconv.ParseFloat(string(res.Data), 64)
	if err != nil {
		return 0.0, err
	}

	return avgSize, nil
}

func GetAvgObjectStoredByUidInDateRange(uid string, from, to time.Time) (int64, error) {
	request := Req{
		Limit:  0,
		Offset: 0,
		Type:   "AvgCount",
		Data:   []string{from.Format(time.RFC3339), to.Format(time.RFC3339), uid},
	}

	jsonReq, _ := json.Marshal(request)

	res, err := nc.Request(fileSubject+"query", jsonReq, contextExpTime)
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(string(res.Data), 10, 64)
}
