package seaweedfs

import (
	"github.com/linxGnu/goseaweedfs"
	"github.com/spf13/viper"
	"net/http"
	"time"
)

var (
	sw      *goseaweedfs.Seaweed
	filer   []string
	swFiler *goseaweedfs.Filer
)

const (
	CHUNK_SIZE = 8096
)

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

func CleanUp() {
	if sw != nil {
		_ = sw.Close()
	}
}
