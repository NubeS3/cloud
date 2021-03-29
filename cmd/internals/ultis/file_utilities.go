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

func StandardizedPath(path string, isBucketFolder bool) string {
	if path == "/" {
		return ""
	}
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

	if isBucketFolder {
		s = "/" + s
	}

	s = s[:len(s)-1]

	return s
}

func GetParentPath(path string) string {
	token := strings.Split(path, "/")

	if len(token) > 0 {
		token = token[:len(token)-1]
	}
	s := "/"
	for _, str := range token {
		if str != "" {
			s += str + "/"
		}
	}

	if s != "/" {
		s = s[:len(s)-1]
	}

	return s
}

func GetBucketName(path string) string {
	if path == "/" {
		return ""
	}

	if path[0] == '/' {
		path = path[1:]
	}

	tokens := strings.Split(path, "/")
	return tokens[0]
}

func GetFileName(path string) string {
	if path == "/" {
		return ""
	}

	if path[0] == '/' {
		path = path[1:]
	}

	tokens := strings.Split(path, "/")
	return tokens[len(tokens)-1]
}
