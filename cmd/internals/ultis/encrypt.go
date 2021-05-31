package ultis

import (
	"crypto/md5"
	"github.com/NubeS3/cloud/cmd/internals/models"
	"github.com/blend/go-sdk/crypto"
	"io"
)

func createHash(key string) []byte {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hasher.Sum(nil)
}

func EncryptReader(src io.Reader, passphrase string) (*crypto.StreamEncrypter, error) {
	master := createHash(passphrase)

	encrypter, err := crypto.NewStreamEncrypter(master, src)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.GeneratorError,
		}
	}

	//if _, err = sio.Encrypt(dst, src, sio.Config{Key: key[:]}); err != nil {
	//	return nil, &models.ModelError{
	//		Msg:     err.Error(),
	//		ErrType: models.GeneratorError,
	//	}
	//}

	return encrypter, nil
}

func DecryptReader(src io.Reader, meta crypto.StreamMeta, passphrase string) (r io.Reader, err error) {
	master := createHash(passphrase)

	decrypter, err := crypto.NewStreamDecrypter(master, meta, src)
	if err != nil {
		return nil, &models.ModelError{
			Msg:     err.Error(),
			ErrType: models.GeneratorError,
		}
	}

	return decrypter, nil
}
