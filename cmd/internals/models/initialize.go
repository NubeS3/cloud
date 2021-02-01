package models

import (
	"net/http"
	"time"

	"github.com/gocql/gocql"
	"github.com/linxGnu/goseaweedfs"
	"github.com/mediocregopher/radix/v3"
	"github.com/spf13/viper"
)

var (
	session     *gocql.Session
	sw          *goseaweedfs.Seaweed
	filer       []string
	swFiler     *goseaweedfs.Filer
	redisClient *radix.Pool
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

	if err := initDbTables(); err != nil {
		return err
	}

	redisClient, err = radix.NewPool("tcp", "127.0.0.1:6379", 10)
	if err != nil {
		return err
	}

	return nil
}

func InitFs() error {
	masterUrl := viper.GetString("SW_MASTER")
	filerUrl := viper.GetString("SW_FILER")

	if _filer := filerUrl; _filer != "" {
		filer = []string{_filer}
	}
	var err error
	sw, err = goseaweedfs.NewSeaweed(masterUrl, filer, CHUNK_SIZE, &http.Client{Timeout: 5 * time.Minute})

	if err != nil {
		return err
	}

	swFiler = sw.Filers()[0]
	return nil
}

func initDbTables() error {
	err := session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" users_by_id (id uuid PRIMARY KEY, username ascii," +
			" password ascii, refresh_token string, is_active boolean)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" users_by_username (id uuid, username ascii PRIMARY KEY," +
			" password ascii, is_active boolean)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" users_by_email (id uuid, username ascii," +
			" password ascii, email ascii PRIMARY KEY," +
			" is_active boolean").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" user_data_by_id (id uuid PRIMARY KEY, email ascii," +
			" gender boolean, company ascii, firstname text," +
			" lastname text, dob date, is_active boolean)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXIST" +
			" user_otp (username ascii PRIMARY KEY, email ascii," +
			" otp ascii, is_validated boolean, last_updated date, expired_time timestamp)").
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
			" (uid uuid, bucket_id uuid, key ascii, permissions set<int>," +
			" expired_date timestamp, PRIMARY KEY ((uid), bucket_id, key))").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" access_keys_by_key" +
			" (uid uuid, bucket_id uuid, key ascii, permissions set<int>," +
			" expired_date timestamp, PRIMARY KEY ((key), bucket_id))").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("create table if not exists folders" +
			" (bucket_id uuid, path text, name text," +
			" primary key ((bucket_id), path, name))" +
			" with clustering order by (path asc, name asc) ").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" file_metadata_by_id" +
			" (id ascii, bucket_id uuid, path text, name text, content_type ascii," +
			" size int, is_hidden boolean, is_deleted boolean, deleted_date timestamp," +
			" upload_date timestamp, expired_date timestamp," +
			" PRIMARY KEY ((id), upload_date, bucket_id))" +
			" with clustering order by (upload_date desc, bucket_id asc)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" file_metadata_by_pathname" +
			" (id ascii, bucket_id uuid, path text, name text, content_type ascii," +
			" size int, is_hidden boolean, is_deleted boolean, deleted_date timestamp," +
			" upload_date timestamp, expired_date timestamp," +
			" PRIMARY KEY ((bucket_id), path, name, upload_date))" +
			" with clustering order by (path asc, name asc, upload_date desc)").
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
	if redisClient != nil {
		_ = redisClient.Close()
	}
}
