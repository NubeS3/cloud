package ultis

import (
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

func GetFileContentType(out multipart.File) (string, error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)

	_, err := out.Read(buffer)
	if err != nil {
		return "", err
	}

	_, err = out.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)

	return contentType, nil
}

func StandardizedPath(path string) (string, error) {
	var s string
	tokenSpace := strings.Fields(path)
	for _, str := range tokenSpace {
		if str != "" {
			s += str
		}
	}

	tokenBackSplash := strings.Split(s, "/")
	s = ""
	for _, str := range tokenBackSplash {
		if str != "" {
			s += str + "/"
		}
	}

	return s, nil
}
