package models

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/gocql/gocql"
	"github.com/spf13/viper"
	"io/ioutil"
	"path/filepath"
	"time"
)

var (
	session  *gocql.Session
	keyspace string
	cqlport  string
)

func IninDb() error {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	cqlshrcHost := fmt.Sprintf("%s-%s.db.astra.datastax.com", viper.GetString("ASTRA_DB_ID"), viper.GetString("ASTRA_DB_REGION"))
	//_cqlshrc_port :=
	_username := viper.GetString("ASTRA_DB_USERNAME")
	_password := viper.GetString("ASTRA_DB_PASSWORD")
	keyspace = viper.GetString("ASTRA_DB_KEYSPACE")
	cqlport = viper.GetString("ASTRA_DB_CQL_PORT")

	_certPath, _ := filepath.Abs("./configs/cert")
	_keyPath, _ := filepath.Abs("./configs/key")
	_caPath, _ := filepath.Abs("./configs/ca.crt")
	_cert, _ := tls.LoadX509KeyPair(_certPath, _keyPath)
	_caCert, _ := ioutil.ReadFile(_caPath)
	_caCertPool := x509.NewCertPool()
	_caCertPool.AppendCertsFromPEM(_caCert)
	_tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{_cert},
		RootCAs:      _caCertPool,
	}

	cluster := gocql.NewCluster(cqlshrcHost)
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: _username,
		Password: _password,
	}
	clusterHosts := fmt.Sprintf("%s:%s", cqlshrcHost, cqlport)
	fmt.Println(clusterHosts)
	cluster.Hosts = []string{clusterHosts}

	cluster.SslOpts = &gocql.SslOptions{
		Config:                 _tlsConfig,
		EnableHostVerification: false,
	}

	cluster.ConnectTimeout = time.Second * 10
	session, err = cluster.CreateSession()
	if err != nil {
		return err
	}

	_query := "SELECT * FROM system.local"
	iter := session.Query(_query).Iter()
	fmt.Printf("Testing: %d rows returned", iter.NumRows())

	return nil
}
