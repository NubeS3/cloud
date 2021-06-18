package arango

import (
	"context"
	arangoDriver "github.com/arangodb/go-driver"
	arangoHttp "github.com/arangodb/go-driver/http"
	scrypt "github.com/elithrar/simple-scrypt"
	"github.com/spf13/viper"
	"time"
)

const CONTEXT_EXPIRED_TIME = 30

var (
	arangoConnection arangoDriver.Connection
	arangoClient     arangoDriver.Client
	arangoDb         arangoDriver.Database
	userCol          arangoDriver.Collection
	otpCol           arangoDriver.Collection
	bucketCol        arangoDriver.Collection
	folderCol        arangoDriver.Collection
	keyPairsCol      arangoDriver.Collection
	apiKeyCol        arangoDriver.Collection
	rfTokenCol       arangoDriver.Collection
	fileMetadataCol  arangoDriver.Collection
	adminCol         arangoDriver.Collection
	bucketSizeCol    arangoDriver.Collection
	encryptCol       arangoDriver.Collection
	snapCol          arangoDriver.Collection
)

func InitArangoDb() error {
	var err error
	//_cqlshrc_port :=
	hostUrl := viper.GetString("ARANGODB_HOST")
	_username := viper.GetString("ARANGODB_USER")
	_password := viper.GetString("ARANGODB_PASSWORD")
	println("connecting to db at" + hostUrl)
	arangoConnection, err = arangoHttp.NewConnection(arangoHttp.ConnectionConfig{
		Endpoints: []string{hostUrl},
	})

	if err != nil {
		panic(err)
	}

	println("creating new client")
	arangoClient, err = arangoDriver.NewClient(arangoDriver.ClientConfig{
		Connection:     arangoConnection,
		Authentication: arangoDriver.BasicAuthentication(_username, _password),
	})

	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	println("creating database")
	dbExist, err := arangoClient.DatabaseExists(ctx, "nubes3")
	if err != nil {
		panic(err)
	}

	if !dbExist {
		arangoDb, err = arangoClient.CreateDatabase(ctx, "nubes3", &arangoDriver.CreateDatabaseOptions{
			Users: []arangoDriver.CreateDatabaseUserOptions{
				{
					UserName: _username,
					Password: _password,
				},
			},
		})

		if err != nil {
			panic(err)
		}
	} else {
		arangoDb, err = arangoClient.Database(ctx, "nubes3")

		if err != nil {
			panic(err)
		}
	}

	return initArangoDb()
}

func initArangoDb() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	exist, err = arangoDb.CollectionExists(ctx, "rfTokens")
	if err != nil {
		return err
	}
	if !exist {
		rfTokenCol, _ = arangoDb.CreateCollection(ctx, "rfTokens", &arangoDriver.CreateCollectionOptions{})
	} else {
		rfTokenCol, _ = arangoDb.Collection(ctx, "rfTokens")
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

	exist, err = arangoDb.CollectionExists(ctx, "folders")
	if err != nil {
		return err
	}
	if !exist {
		folderCol, _ = arangoDb.CreateCollection(ctx, "folders", &arangoDriver.CreateCollectionOptions{})
	} else {
		folderCol, _ = arangoDb.Collection(ctx, "folders")
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

	//exist, err = arangoDb.CollectionExists(ctx, "keyPairs")
	//if err != nil {
	//	return err
	//}
	//if !exist {
	//	keyPairsCol, _ = arangoDb.CreateCollection(ctx, "keyPairs", &arangoDriver.CreateCollectionOptions{})
	//} else {
	//	keyPairsCol, _ = arangoDb.Collection(ctx, "keyPairs")
	//}

	exist, err = arangoDb.CollectionExists(ctx, "fileMetadata")
	if err != nil {
		return err
	}
	if !exist {
		fileMetadataCol, _ = arangoDb.CreateCollection(ctx, "fileMetadata", &arangoDriver.CreateCollectionOptions{})
	} else {
		fileMetadataCol, _ = arangoDb.Collection(ctx, "fileMetadata")
	}

	exist, err = arangoDb.CollectionExists(ctx, "bucketSize")
	if err != nil {
		return err
	}
	if !exist {
		bucketSizeCol, _ = arangoDb.CreateCollection(ctx, "bucketSize", &arangoDriver.CreateCollectionOptions{})
	} else {
		bucketSizeCol, _ = arangoDb.Collection(ctx, "bucketSize")
	}

	exist, err = arangoDb.CollectionExists(ctx, "encryptInfo")
	if err != nil {
		return err
	}
	if !exist {
		encryptCol, _ = arangoDb.CreateCollection(ctx, "encryptInfo", &arangoDriver.CreateCollectionOptions{})
	} else {
		encryptCol, _ = arangoDb.Collection(ctx, "encryptInfo")
	}

	exist, err = arangoDb.CollectionExists(ctx, "admin")
	if err != nil {
		return err
	}
	if !exist {
		adminCol, _ = arangoDb.CreateCollection(ctx, "admin", &arangoDriver.CreateCollectionOptions{})
	} else {
		adminCol, _ = arangoDb.Collection(ctx, "admin")
	}

	exist, err = arangoDb.CollectionExists(ctx, "snapshots")
	if err != nil {
		return err
	}
	if !exist {
		snapCol, _ = arangoDb.CreateCollection(ctx, "snapshots", &arangoDriver.CreateCollectionOptions{})
	} else {
		snapCol, _ = arangoDb.Collection(ctx, "snapshots")
	}

	println("initializing admin")
	initAdmin()

	return nil
}

func initAdmin() {
	adminUsername := viper.GetString("ADMIN_ROOT_USERNAME")
	adminPassword := viper.GetString("ADMIN_ROOT_PASSWORD")
	passwordHashed, err := scrypt.GenerateFromPassword([]byte(adminPassword), scrypt.DefaultParams)
	if err != nil {
		panic(err)
	}

	query := "UPSERT { username: @u } " +
		"INSERT { username: @u, password: @p, is_disabled: false, type: @t, created_at: @time, updated_at: @time } " +
		"UPDATE {} " +
		"IN admin"
	bindVars := map[string]interface{}{
		"u":    adminUsername,
		"p":    string(passwordHashed),
		"t":    RootAdmin,
		"time": time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = arangoDb.Query(ctx, query, bindVars)
	if err != nil {
		panic(err)
	}
}
