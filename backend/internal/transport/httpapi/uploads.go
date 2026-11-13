package httpapi

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"unicode/utf8"

	"bashkir-school/backend/internal/domain"
	"bashkir-school/backend/internal/service"

	"github.com/gin-gonic/gin"
)

func readLiteratureForm(c *gin.Context) (string, error) {
	if c.Request.ContentLength > domain.MaxLiteratureBytes+16384 {
		return "", fmt.Errorf("файл слишком большой")
	}
	fh, err := c.FormFile("file")
	if err == nil {
		if fh.Size > domain.MaxLiteratureBytes {
			return "", fmt.Errorf("размер TXT не должен превышать 1 МБ")
		}
		var f multipart.File
		f, err = fh.Open()
		if err != nil {
			return "", err
		}
		defer f.Close()
		b, readErr := io.ReadAll(io.LimitReader(f, domain.MaxLiteratureBytes+1))
		if readErr != nil {
			return "", readErr
		}
		if len(b) > domain.MaxLiteratureBytes {
			return "", fmt.Errorf("файл слишком большой")
		}
		if !strings.HasSuffix(strings.ToLower(fh.Filename), ".txt") {
			return "", fmt.Errorf("поддерживается только TXT")
		}
		if !utf8.Valid(b) {
			return "", fmt.Errorf("нужен TXT в кодировке UTF-8")
		}
		return string(bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})), nil
	}
	text := c.PostForm("text")
	if len(text) > domain.MaxLiteratureBytes || !utf8.ValidString(text) {
		return "", fmt.Errorf("текст слишком большой или не UTF-8")
	}
	return text, nil
}

func uploadFields(c *gin.Context) service.UploadFields {
	fields := service.UploadFields{}
	_ = c.Request.ParseMultipartForm(1 << 20)
	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			fields[key] = values[0]
		}
	}
	return fields
}
func readLiteratureUpload(c *gin.Context) (service.LiteratureUpload, error) {
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()
	text, err := readLiteratureForm(c)
	fields := uploadFields(c)
	return service.LiteratureUpload{UploadFields: fields, Text: text, ReadError: err}, nil
}
func readAudioUpload(c *gin.Context) (service.AudioUpload, error) {
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()
	file, err := c.FormFile("audio")
	fields := uploadFields(c)
	if err != nil {
		return service.AudioUpload{UploadFields: fields, ReadError: err}, nil
	}
	if file.Size > domain.MaxAudioBytes {
		return service.AudioUpload{UploadFields: fields, ReadError: fmt.Errorf("аудио больше 20 МБ")}, nil
	}
	f, err := file.Open()
	if err != nil {
		return service.AudioUpload{UploadFields: fields, ReadError: err}, nil
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, domain.MaxAudioBytes+1))
	return service.AudioUpload{UploadFields: fields, Audio: data, ContentType: http.DetectContentType(data), Filename: file.Filename, ReadError: err}, nil
}
