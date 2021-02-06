package ultis

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/viper"
)

var (
	sgSender string
	sgEmail string
)

func InitMailService() {
	sgSender = viper.GetString("SG_SENDER")
	sgEmail = viper.GetString("SG_EMAIL")
}

func SendMail(receiver string, receiverEmail string, subject string, content string) error {
	from := mail.NewEmail(sgSender, sgEmail)
	to := mail.NewEmail(receiver, receiverEmail)
	message := mail.NewSingleEmail(from, subject, to, content, "")
	client := sendgrid.NewSendClient(viper.GetString("SG_KEY"))
	_, err := client.Send(message)

	return err
}