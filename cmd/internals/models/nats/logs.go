package nats

import (
	"encoding/json"
	"time"
)

type Event struct {
	Type string    `json:"type"`
	Date time.Time `json:"event_time"`
}

type errLogMessage struct {
	Content string    `json:"content"`
	Type    string    `json:"type"`
	At      time.Time `json:"at"`
}

func SendErrorEvent(content, t string) error {
	jsonData, err := json.Marshal(errLogMessage{
		Content: content,
		Type:    t,
		At:      time.Now(),
	})

	if err != nil {
		return err
	}

	return sc.Publish(errSubj, jsonData)
}

type fileLog struct {
	Id          string    `json:"id"`
	FId         string    `json:"f_id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	BucketId    string    `json:"bucket_id"`
	ContentType string    `json:"content_type"`
	UploadDate  time.Time `json:"upload_date"`
	Path        string    `json:"path"`
	IsHidden    bool      `json:"is_hidden"`
}

func SendUploadFileEvent(id, fid, name string, size int64,
	bid, contentType string, uploadDate time.Time, path string, isHidden bool) error {
	jsonData, err := json.Marshal(fileLog{
		Id:          id,
		FId:         fid,
		Name:        name,
		Size:        size,
		BucketId:    bid,
		ContentType: contentType,
		UploadDate:  uploadDate,
		Path:        path,
		IsHidden:    isHidden,
	})

	if err != nil {
		return err
	}

	return sc.Publish(uploadFileSubj, jsonData)
}

func SendDownloadFileEvent(id, fid, name string, size int64,
	bid, contentType string, uploadDate time.Time, path string, isHidden bool) error {
	jsonData, err := json.Marshal(fileLog{
		Id:          id,
		FId:         fid,
		Name:        name,
		Size:        size,
		BucketId:    bid,
		ContentType: contentType,
		UploadDate:  uploadDate,
		Path:        path,
		IsHidden:    isHidden,
	})

	if err != nil {
		return err
	}

	return sc.Publish(downloadFileSubj, jsonData)
}

type stagingFileLog struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	BucketId    string    `json:"bucket_id"`
	ContentType string    `json:"content_type"`
	UploadDate  time.Time `json:"upload_date"`
	Path        string    `json:"path"`
	IsHidden    bool      `json:"is_hidden"`
}

func SendStagingFileEvent(name string, size int64, bid, contentType, path string, isHidden bool) error {
	jsonData, err := json.Marshal(stagingFileLog{
		Name:        name,
		Size:        size,
		BucketId:    bid,
		ContentType: contentType,
		UploadDate:  time.Now(),
		Path:        path,
		IsHidden:    isHidden,
	})

	if err != nil {
		return err
	}

	return sc.Publish(stagingFileSubj, jsonData)
}

type uploadSuccessFileLog struct {
	Id          string    `json:"id"`
	FId         string    `json:"f_id"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	BucketId    string    `json:"bucket_id"`
	ContentType string    `json:"content_type"`
	UploadDate  time.Time `json:"upload_date"`
	Path        string    `json:"path"`
	IsHidden    bool      `json:"is_hidden"`
}

func SendUploadSuccessFileEvent(id, fid, name string, size int64,
	bid, contentType string, uploadDate time.Time, path string, isHidden bool) error {
	jsonData, err := json.Marshal(uploadSuccessFileLog{
		Id:          id,
		FId:         fid,
		Name:        name,
		Size:        size,
		BucketId:    bid,
		ContentType: contentType,
		UploadDate:  uploadDate,
		Path:        path,
		IsHidden:    isHidden,
	})

	if err != nil {
		return err
	}

	return sc.Publish(uploadFileSuccessSubj, jsonData)
}

type UserEvent struct {
	EventLog Event `json:"event_log"`

	Firstname string    `json:"firstname"`
	Lastname  string    `json:"lastname"`
	Username  string    `json:"username"`
	Pass      string    `json:"password"`
	Email     string    `json:"email"`
	Dob       time.Time `json:"dob"`
	Company   string    `json:"company"`
	Gender    bool      `json:"gender"`
	IsActive  bool      `json:"is_active"`
	IsBanned  bool      `json:"is_banned"`
}

func SendUserEvent(firstname, lastname, username, pass, email string,
	dob time.Time, company string, gender, isActive, isBanned bool, eventType string) error {
	jsonData, err := json.Marshal(UserEvent{
		EventLog: Event{
			Type: eventType,
			Date: time.Now(),
		},
		Firstname: firstname,
		Lastname:  lastname,
		Username:  username,
		Pass:      pass,
		Email:     email,
		Dob:       dob,
		Company:   company,
		Gender:    gender,
		IsActive:  isActive,
		IsBanned:  isBanned,
	})

	if err != nil {
		return err
	}

	return sc.Publish(userSubj, jsonData)
}

type BucketEvent struct {
	EventLog Event `json:"event_log"`

	Id     string `json:"id"`
	Uid    string `json:"uid"`
	Name   string `json:"name"`
	Region string `json:"region"`
}

func SendBucketEvent(id, uid, name, region, eventType string) error {
	jsonData, err := json.Marshal(BucketEvent{
		EventLog: Event{
			Type: eventType,
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

	return sc.Publish(bucketSubj, jsonData)
}

type FolderEvent struct {
	EventLog Event `json:"event_log"`

	Id       string `json:"-"`
	OwnerId  string `json:"owner_id"`
	Name     string `json:"name"`
	Fullpath string `json:"fullpath"`
}

func SendFolderEvent(id, ownerId, name, fullpath, eventType string) error {
	jsonData, err := json.Marshal(FolderEvent{
		EventLog: Event{
			Type: eventType,
			Date: time.Now(),
		},
		Id:       id,
		OwnerId:  ownerId,
		Name:     name,
		Fullpath: fullpath,
	})

	if err != nil {
		return err
	}

	return sc.Publish(folderSubj, jsonData)
}

type AccessKeyLog struct {
	EventLog Event `json:"event_log"`

	Key         string    `json:"key"`
	BucketId    string    `json:"bucket_id"`
	ExpiredDate time.Time `json:"expired_date"`
	Permissions []string  `json:"permissions"`
	Uid         string    `json:"uid"`
}

func SendAccessKeyEvent(key, bid string, expiredDate time.Time,
	permissions []string, uid string, eventType string) error {
	jsonData, err := json.Marshal(AccessKeyLog{
		EventLog: Event{
			Type: eventType,
			Date: time.Now(),
		},
		Key:         key,
		BucketId:    bid,
		ExpiredDate: expiredDate,
		Permissions: permissions,
		Uid:         uid,
	})

	if err != nil {
		return err
	}

	return sc.Publish(accessKeySubj, jsonData)
}

type KeyPairLog struct {
	EventLog Event `json:"event_log"`

	Public       string    `json:"public"`
	Private      string    `json:"private"`
	BucketId     string    `json:"bucket_id"`
	GeneratorUid string    `json:"generator_uid"`
	ExpiredDate  time.Time `json:"expired_date"`
	Permissions  []string  `json:"permissions"`
}

func SendKeyPairEvent(public, private, bid string, expiredDate time.Time,
	permissions []string, uid string, eventType string) error {
	jsonData, err := json.Marshal(KeyPairLog{
		EventLog: Event{
			Type: eventType,
			Date: time.Now(),
		},
		Public:       public,
		Private:      private,
		BucketId:     bid,
		GeneratorUid: uid,
		ExpiredDate:  expiredDate,
		Permissions:  permissions,
	})

	if err != nil {
		return err
	}

	return sc.Publish(keyPairSubj, jsonData)
}

type AdminLog struct {
	EventLog Event  `json:"event_log"`
	AdminId  string `json:"admin_id"`
	Admin    string `json:"admin"`
	Content  string `json:"content"`
}

func SendAdminEvent(id, admin, content string) error {
	jsonData, err := json.Marshal(AdminLog{
		EventLog: Event{
			Type: "Admin",
			Date: time.Now(),
		},
		AdminId: id,
		Admin:   admin,
		Content: content,
	})

	if err != nil {
		return err
	}

	return sc.Publish(keyPairSubj, jsonData)
}
