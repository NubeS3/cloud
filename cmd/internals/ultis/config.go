package ultis

import "github.com/spf13/viper"

var secret string

func InitUtilities() {
	secret = viper.GetString("SECRET")
}
