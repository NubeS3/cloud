package cassandra

import (
	"time"

	"github.com/gocql/gocql"
	"github.com/mediocregopher/radix/v3"
	"github.com/spf13/viper"
)

var (
	session     *gocql.Session
	redisClient *radix.Pool
)

func InitCassandraDb() error {
	var err error
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
	cluster.Consistency = gocql.One

	cluster.ConnectTimeout = time.Second * 10
	session, err = cluster.CreateSession()
	if err != nil {
		return err
	}

	if err := initCassandraDbTables(); err != nil {
		return err
	}

	return nil
}

func initCassandraDbTables() error {
	err := session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" errors_log (id, type ascii," +
			" message text, time timestamp, PRIMARY KEY ((id), type, time))").
		Exec()
	if err != nil {
		return err
	}

	return nil
}

func CleanUp() {
	if session != nil {
		session.Close()
	}
}
