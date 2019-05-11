package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"

	"github.com/jasonmajors/media-upload/backblaze"
)

const maxUploadSize = 2 * 1024 * 1024 * 5 // 10mb

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

func getFileBytes(r *http.Request, key string) <-chan backblaze.UploadFile {
	// need to return the handler...
	file, handler := getFileFromForm(r, key)
	bytesOut := make(chan backblaze.UploadFile)

	defer file.Close()

	go func(handler *multipart.FileHeader) {
		log.Println("Detecting and validating filetype for: ", handler.Filename)

		fileBytes := getBytes(file)
		fileType := http.DetectContentType(fileBytes)
		// Check valid mimetype
		if valid := validateFileType(fileType); valid != true {
			panic("getFileBytes: Invalid file type")
		}
		uploadFile := new(backblaze.UploadFile)
		uploadFile.Handler = handler
		uploadFile.Bytes = fileBytes

		bytesOut <- *uploadFile

		close(bytesOut)
	}(handler)

	return bytesOut
}

func Upload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	if r.Method == "POST" {
		// Check file size isnt too big
		if ok := fileSizeIsOk(w, r); ok != true {
			log.Println("File too big")
			jsonErr(w, "File too big, man", http.StatusBadRequest)
			return
		}
		imageBytes := getFileBytes(r, "image")
		audioBytes := getFileBytes(r, "audio")

		imageFileBytes := <-imageBytes
		audioFileBytes := <-audioBytes

		payload := []backblaze.UploadFile{imageFileBytes, audioFileBytes}
		responses, err := backblaze.Save(payload)
		if err != nil {
			fmt.Println(err)
			return
		}

		response := make(map[string]string)

		for _, backblazeResp := range responses {
			response[backblazeResp.ApiResponse.FileName] = backblazeResp.DownloadUrl
			log.Println("Download URL: ", backblazeResp.DownloadUrl)
		}
		jsonResp, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(jsonResp)
		return
	} else {
		log.Println("Method not allowed")
		jsonErr(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func jsonErr(w http.ResponseWriter, message string, status int) {
	w.WriteHeader(status)

	errResp := make(map[string]string)
	errResp["error"] = message
	jsonErr, _ := json.Marshal(errResp)

	w.Write(jsonErr)
}

func main() {
	http.HandleFunc("/upload", Upload)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
