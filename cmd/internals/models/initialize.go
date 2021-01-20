package models

import (
	"github.com/gocql/gocql"
	"github.com/linxGnu/goseaweedfs"
	"github.com/spf13/viper"
	"net/http"
	"time"
)

var (
	session *gocql.Session
	sw      *goseaweedfs.Seaweed
)

const (
	CHUNK_SIZE = 8096
)

func InitDb() error {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	//_cqlshrc_port :=
	cqlshrcHost := viper.GetString("DB_URL")
	_username := viper.GetString("DB_USERNAME")
	_password := viper.GetString("DB_PASSWORD")
	keyspace := viper.GetString("DB_KEYSPACE")

	cluster := gocql.NewCluster(cqlshrcHost)
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: _username,
		Password: _password,
	}
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum

	cluster.ConnectTimeout = time.Second * 10
	session, err = cluster.CreateSession()
	if err != nil {
		return err
	}

	//_query := "SELECT * FROM system.local"
	//iter := session.Query(_query).Iter()
	//fmt.Printf("Testing: %d rows returned", iter.NumRows())

	return nil
}

func InitFs() error {
	masterUrl := viper.GetString("SW_MASTER")
	filerUrl := viper.GetString("SW_FILER")

	var err error
	sw, err = goseaweedfs.NewSeaweed(masterUrl, []string{filerUrl}, CHUNK_SIZE, &http.Client{Timeout: 5 * time.Minute})

	return err
}

func CleanUp() {
	if sw != nil {
		_ = sw.Close()
	}
	if session != nil {
		session.Close()
	}
}
