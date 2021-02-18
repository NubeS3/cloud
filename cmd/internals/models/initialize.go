package models

import (
	"context"
	"net/http"
	"time"

	arangoDriver "github.com/arangodb/go-driver"
	arangoHttp "github.com/arangodb/go-driver/http"
	"github.com/gocql/gocql"
	"github.com/linxGnu/goseaweedfs"
	"github.com/mediocregopher/radix/v3"
	"github.com/spf13/viper"
)

var (
	session          *gocql.Session
	sw               *goseaweedfs.Seaweed
	filer            []string
	swFiler          *goseaweedfs.Filer
	redisClient      *radix.Pool
	arangoConnection arangoDriver.Connection
	arangoClient     arangoDriver.Client
	arangoDb         arangoDriver.Database
)

const (
	CHUNK_SIZE = 8096
)

func InitArangoDb() error {
	var err error
	//_cqlshrc_port :=
	hostUrl := viper.GetString("ARANGODB_HOST")
	_username := viper.GetString("ARANGODB_USER")
	_password := viper.GetString("ARANGODB_PASSWORD")
	arangoConnection, err = arangoHttp.NewConnection(arangoHttp.ConnectionConfig{
		Endpoints: []string{hostUrl},
	})

	if err != nil {
		return err
	}

	arangoClient, err = arangoDriver.NewClient(arangoDriver.ClientConfig{
		Connection:     arangoConnection,
		Authentication: arangoDriver.BasicAuthentication(_username, _password),
	})

	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbExist, err := arangoClient.DatabaseExists(ctx, "nubes3")
	if err != nil {
		return err
	}

	if !dbExist {
		arangoDb, _ = arangoClient.CreateDatabase(ctx, "nubes3", &arangoDriver.CreateDatabaseOptions{
			Users: []arangoDriver.CreateDatabaseUserOptions{
				{
					UserName: _username,
					Password: _password,
				},
			},
		})
	} else {
		arangoDb, _ = arangoClient.Database(ctx, "nubes3")
	}

	return initArangoDb()
}

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

	//_query := "SELECT * FROM system.local"
	//iter := session.Query(_query).Iter()
	//fmt.Printf("Testing: %d rows returned", iter.NumRows())

	if err := initCassandraDbTables(); err != nil {
		return err
	}

	//redisClient, err = radix.NewPool("tcp", "127.0.0.1:6379", 10)
	//if err != nil {
	//	return err
	//}

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

func initArangoDb() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// INIT DB
	_, _ = arangoDb.CreateCollection(ctx, "users", &arangoDriver.CreateCollectionOptions{})
	//_, _ = arangoDb.CreateCollection(ctx, "userHasBuckets", &arangoDriver.CreateCollectionOptions{})
	_, _ = arangoDb.CreateCollection(ctx, "buckets", &arangoDriver.CreateCollectionOptions{})
	//_, _ = arangoDb.CreateCollection(ctx, "bucketHasApiKeys", &arangoDriver.CreateCollectionOptions{})
	_, _ = arangoDb.CreateCollection(ctx, "apiKeys", &arangoDriver.CreateCollectionOptions{})
	//_, _ = arangoDb.CreateCollection(ctx, "bucketHasFileMetadata", &arangoDriver.CreateCollectionOptions{})
	_, _ = arangoDb.CreateCollection(ctx, "fileMetadata", &arangoDriver.CreateCollectionOptions{})
	// INIT GRAPH
	edgeDefinition := arangoDriver.EdgeDefinition{
		Collection: "userHasBuckets",
		To:         []string{"buckets"},
		From:       []string{"users"},
	}
	_, _ = arangoDb.CreateGraph(nil, "usersBuckets", &arangoDriver.CreateGraphOptions{
		EdgeDefinitions: []arangoDriver.EdgeDefinition{edgeDefinition},
	})

	edgeDefinition = arangoDriver.EdgeDefinition{
		Collection: "bucketHasApiKeys",
		To:         []string{"apiKeys"},
		From:       []string{"buckets"},
	}
	_, _ = arangoDb.CreateGraph(nil, "usersBuckets", &arangoDriver.CreateGraphOptions{
		EdgeDefinitions: []arangoDriver.EdgeDefinition{edgeDefinition},
	})

	edgeDefinition = arangoDriver.EdgeDefinition{
		Collection: "bucketHasFileMetadata",
		To:         []string{"fileMetadata"},
		From:       []string{"buckets"},
	}
	_, _ = arangoDb.CreateGraph(nil, "bucketsMetadata", &arangoDriver.CreateGraphOptions{
		EdgeDefinitions: []arangoDriver.EdgeDefinition{edgeDefinition},
	})

	return nil
}

func initCassandraDbTables() error {
	err := session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" users_by_id (id uuid PRIMARY KEY, username ascii," +
			" password ascii, " +
			" is_active boolean, is_banned boolean)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" users_by_username (id uuid, username ascii PRIMARY KEY," +
			" password ascii, is_active boolean, is_banned boolean)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" users_by_email (id uuid, username ascii," +
			" password ascii, email ascii PRIMARY KEY," +
			" is_active boolean, is_banned boolean)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" user_data_by_id (id uuid PRIMARY KEY, email ascii," +
			" gender boolean, company ascii, firstname text," +
			" lastname text, dob date, is_active boolean," +
			" created_at timestamp, updated_at timestamp," +
			" is_banned boolean)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" user_otp (username ascii PRIMARY KEY, id uuid, email ascii," +
			" otp ascii, last_updated timestamp, expired_time timestamp, is_validated boolean)").
		Exec()
	if err != nil {
		return err
	}

	err = session.
		Query("CREATE TABLE IF NOT EXISTS" +
			" refresh_token (uid uuid, rf_token ascii, expired_rf timestamp," +
			" PRIMARY KEY (uid))").
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
			" PRIMARY KEY ((id), bucket_id, upload_date))" +
			" with clustering order by (bucket_id asc, upload_date desc)").
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
