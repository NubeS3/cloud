package ultis

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/viper"
)

func SendMail(receiver string, receiverEmail string, subject string, content string) error {
	sender := viper.GetString("SG_SENDER")
	senderEmail := viper.GetString("SG_EMAIL")
	from := mail.NewEmail(sender, senderEmail)
	to := mail.NewEmail(receiver, receiverEmail)
	message := mail.NewSingleEmail(from, subject, to, content, "")
	client := sendgrid.NewSendClient(viper.GetString("SG_KEY"))
	_, err := client.Send(message)

	return err
}