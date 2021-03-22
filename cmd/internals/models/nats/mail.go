package nats

import "time"

type mailMessage struct {
	Otp      string    `json:"otp"`
	Username string    `json:"username"`
	To       string    `json:"to"`
	Exp      time.Time `json:"exp"`
}

func SendEmailEvent(email, username, otp string, expired time.Time) error {
	return c.Publish(mailSubj, &mailMessage{
		Otp:      otp,
		Username: username,
		To:       email,
		Exp:      expired,
	})
}
