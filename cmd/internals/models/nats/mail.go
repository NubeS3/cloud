package nats

import (
	"encoding/json"
	"time"
)

type mailMessage struct {
	Otp string    `json:"otp"`
	To  string    `json:"to"`
	Exp time.Time `json:"exp"`
}

func SendEmailEvent(email, otp string, expired time.Time) error {
	jsonData, err := json.Marshal(mailMessage{
		Otp: otp,
		To:  email,
		Exp: expired,
	})

	if err != nil {
		return err
	}
	//return sc.Publish(mailSubj, jsonData)
	_, err = js.Publish("NUBES3."+mailSubj, jsonData)
	return err
}
