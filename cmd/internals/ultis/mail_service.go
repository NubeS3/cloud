package ultis

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/viper"
)

var (
	sgSender string
	sgEmail  string
	sgKey    string
)

func InitMailService() {
	sgSender = viper.GetString("SG_SENDER")
	sgEmail = viper.GetString("SG_EMAIL")
	sgKey = viper.GetString("SG_KEY")
}

func SendMail(receiver string, receiverEmail string, subject string, content string) error {
	from := mail.NewEmail(sgSender, sgEmail)
	to := mail.NewEmail(receiver, receiverEmail)
	message := mail.NewSingleEmail(from, subject, to, content, "")
	client := sendgrid.NewSendClient(sgKey)
	_, err := client.Send(message)

	return err

}
