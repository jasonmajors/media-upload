package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/jasonmajors/media-upload/backblaze"
)

const maxUploadSize = 2 * 1024 * 1024 // 2mb

type UploadFile struct {
	Bytes   []byte
	Handler *multipart.FileHeader
}

func fileSizeIsOk(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		return false
	}
	return true
}

func validateFileType(fileType string) bool {
	var valid bool
	// A case for all valid mimetypes
	switch fileType {
	case "image/jpeg", "image/jpg":
		valid = true
	case "audio/mpeg":
		valid = true
	case "image/png":
		valid = true
	default:
		valid = false
	}
	return valid
}

func getFileFromForm(r *http.Request, key string) (multipart.File, *multipart.FileHeader) {
	file, handler, err := r.FormFile(key)
	if err != nil {
		fmt.Println("Unable to read file", err)
		panic("Unable to read file")
	}
	defer file.Close()

	return file, handler
}

func getBytes(file multipart.File) []byte {
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}

	return fileBytes
}

func getFileBytes(r *http.Request, key string) <-chan UploadFile {
	// need to return the handler...
	file, handler := getFileFromForm(r, key)
	bytesOut := make(chan UploadFile)

	defer file.Close()

	go func(handler *multipart.FileHeader) {
		fmt.Println("gettin dem bytes")

		fileBytes := getBytes(file)
		fileType := http.DetectContentType(fileBytes)
		// Check valid mimetype
		if valid := validateFileType(fileType); valid != true {
			panic("getFileBytes: Invalid file type")
		}
		uploadFile := new(UploadFile)
		uploadFile.Handler = handler
		uploadFile.Bytes = fileBytes

		bytesOut <- *uploadFile

		close(bytesOut)
	}(handler)

	return bytesOut
}

func Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Check file size isnt too big
		if ok := fileSizeIsOk(w, r); ok != true {
			fmt.Fprintf(w, "The file's too big man")
			return
		}
		bytesOut := getFileBytes(r, "image")
		uploadFile := <-bytesOut

		backblaze.Save(w, uploadFile.Bytes, uploadFile.Handler)
	} else {
		fmt.Fprintf(w, "Method not allowed")
	}
}

func main() {
	http.HandleFunc("/upload", Upload)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
