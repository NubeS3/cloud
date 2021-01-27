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
	filer   []string
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
	cluster.Consistency = gocql.One

	cluster.ConnectTimeout = time.Second * 10
	session, err = cluster.CreateSession()
	if err != nil {
		return err
	}

	//_query := "SELECT * FROM system.local"
	//iter := session.Query(_query).Iter()
	//fmt.Printf("Testing: %d rows returned", iter.NumRows())

	return initDbTables()
}

func InitFs() error {
	masterUrl := viper.GetString("SW_MASTER")
	filerUrl := viper.GetString("SW_FILER")

	if _filer := filerUrl; _filer != "" {
		filer = []string{_filer}
	}
	var err error
	sw, err = goseaweedfs.NewSeaweed(masterUrl, filer, CHUNK_SIZE, &http.Client{Timeout: 5 * time.Minute})

	return err
}

func initDbTables() error {
	err := session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" users_by_id (id uuid PRIMARY KEY, username ascii, password ascii)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" user_by_username (id uuid, username ascii PRIMARY KEY," +
			" password ascii)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" user_data_by_id (id uuid PRIMARY KEY, email ascii," +
			" gender boolean, company ascii, firstname text," +
			" lastname text, dob date)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" buckets (id uuid, uid uuid, name ascii, region ascii," +
			" PRIMARY KEY ((uid), id))").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" access_keys_by_uid_bid" +
			" (uid uuid, bucket_id uuid, key ascii, type int," +
			" expired_date date, PRIMARY KEY ((uid), bucket_id, key))").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" access_keys_by_key" +
			" (uid uuid, bucket_id uuid, key ascii, type int," +
			" expired_date date, PRIMARY KEY ((key), bucket_id))").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" file_metadata_by_bid" +
			" (id uuid, bucket_id uuid, name text, type ascii," +
			" length int, upload_date timestamp," +
			" PRIMARY KEY ((bucket_id), upload_date, id))" +
			" with clustering order by (upload_date desc, id asc)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" file_metadata_by_id" +
			" (id uuid, bucket_id uuid, name text, type ascii," +
			" length int, upload_date timestamp," +
			" PRIMARY KEY ((id), upload_date, bucket_id))" +
			" with clustering order by (upload_date desc, bucket_id asc)").
		Exec()
	if err != nil {
		return err
	}

	return nil
}

func CleanUp() {
	if sw != nil {
		_ = sw.Close()
	}
	if session != nil {
		session.Close()
	}
}
