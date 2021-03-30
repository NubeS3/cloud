package nats

import (
	"encoding/json"
	"time"
)

type mailMessage struct {
	Otp      string    `json:"otp"`
	Username string    `json:"username"`
	To       string    `json:"to"`
	Exp      time.Time `json:"exp"`
}

func SendEmailEvent(email, username, otp string, expired time.Time) error {
	jsonData, err := json.Marshal(mailMessage{
		Otp:      otp,
		Username: username,
		To:       email,
		Exp:      expired,
	})

	if err != nil {
		return err
	}
	return sc.Publish(mailSubj, jsonData)
}
