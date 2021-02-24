package arango

import (
	"context"
	arangoDriver "github.com/arangodb/go-driver"
	arangoHttp "github.com/arangodb/go-driver/http"
	"github.com/spf13/viper"
	"time"
)

var (
	arangoConnection arangoDriver.Connection
	arangoClient     arangoDriver.Client
	arangoDb         arangoDriver.Database
	userCol          arangoDriver.Collection
	otpCol           arangoDriver.Collection
	bucketCol        arangoDriver.Collection
	apiKeyCol        arangoDriver.Collection
	fileMetadataCol  arangoDriver.Collection
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

func initArangoDb() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var exist bool
	var err error
	// INIT DB
	exist, err = arangoDb.CollectionExists(ctx, "users")
	if err != nil {
		return err
	}
	if !exist {
		userCol, _ = arangoDb.CreateCollection(ctx, "users", &arangoDriver.CreateCollectionOptions{})
	} else {
		userCol, _ = arangoDb.Collection(ctx, "users")
	}

	exist, err = arangoDb.CollectionExists(ctx, "otps")
	if err != nil {
		return err
	}
	if !exist {
		otpCol, _ = arangoDb.CreateCollection(ctx, "otps", &arangoDriver.CreateCollectionOptions{})
	} else {
		otpCol, _ = arangoDb.Collection(ctx, "otps")
	}

	exist, err = arangoDb.CollectionExists(ctx, "buckets")
	if err != nil {
		return err
	}
	if !exist {
		bucketCol, _ = arangoDb.CreateCollection(ctx, "buckets", &arangoDriver.CreateCollectionOptions{})
	} else {
		bucketCol, _ = arangoDb.Collection(ctx, "buckets")
	}

	exist, err = arangoDb.CollectionExists(ctx, "apiKeys")
	if err != nil {
		return err
	}
	if !exist {
		apiKeyCol, _ = arangoDb.CreateCollection(ctx, "apiKeys", &arangoDriver.CreateCollectionOptions{})
	} else {
		apiKeyCol, _ = arangoDb.Collection(ctx, "apiKeys")
	}

	exist, err = arangoDb.CollectionExists(ctx, "fileMetadata")
	if err != nil {
		return err
	}
	if !exist {
		fileMetadataCol, _ = arangoDb.CreateCollection(ctx, "fileMetadata", &arangoDriver.CreateCollectionOptions{})
	} else {
		fileMetadataCol, _ = arangoDb.Collection(ctx, "fileMetadata")
	}

	return nil
}
