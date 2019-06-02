package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/jasonmajors/media-upload/backblaze"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

const maxUploadSize = 2 * 1024 * 1024 * 5 // 10mb

func fileSizeIsOk(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		log.Println(err.Error())
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
		log.Println("Unable to read file", err)
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
	defer file.Close()

	bytesOut := make(chan backblaze.UploadFile)

	go func(handler *multipart.FileHeader) {
		log.Println("Detecting and validating filetype for: ", handler.Filename)
		fileBytes := getBytes(file)
		fileType := http.DetectContentType(fileBytes)
		uploadFile := new(backblaze.UploadFile)
		// Check valid mimetype
		if valid := validateFileType(fileType); valid == true {
			uploadFile.Handler = handler
			uploadFile.Bytes = fileBytes
		} else {
			uploadFile.Error = errors.New("Invalid file type")
		}
		bytesOut <- *uploadFile
		close(bytesOut)
	}(handler)

	return bytesOut
}

func preparePayloadFromForm(keys []string, w http.ResponseWriter, r *http.Request) ([]backblaze.UploadFile, error) {
	var uploadFileChannels []<-chan backblaze.UploadFile
	var payload []backblaze.UploadFile
	var err error

	for _, formKey := range keys {
		fileChan := getFileBytes(r, formKey)
		uploadFileChannels = append(uploadFileChannels, fileChan)
	}
	for _, fileChan := range uploadFileChannels {
		uploadFile := <-fileChan
		if uploadFile.Error != nil {
			log.Println("Error:", uploadFile.Error.Error())
			err = uploadFile.Error
		}
		payload = append(payload, uploadFile)
	}

	return payload, err
}

func Upload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	if r.Method == "OPTIONS" {
		return
	}
	// Check file size isnt too big
	if ok := fileSizeIsOk(w, r); ok != true {
		log.Println("File too big")
		jsonErr(w, "File too big, man", http.StatusBadRequest)
		return
	}
	// TODO: Real auth
	if secret := r.URL.Query().Get("token"); secret != os.Getenv("TOKEN") {
		log.Println("Unauthorized")
		jsonErr(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method == "POST" {
		formKeys := []string{"image", "audio"}
		payload, err := preparePayloadFromForm(formKeys, w, r)
		if err != nil {
			jsonErr(w, err.Error(), http.StatusBadRequest)
			return
		}

		responses, err := backblaze.Save(payload)
		if err != nil {
			log.Println(err.Error())
			jsonErr(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := make(map[string]string)

		for _, backblazeResp := range responses {
			response[backblazeResp.ApiResponse.FileName] = backblazeResp.DownloadUrl
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
	log.Println("Making json err response")
	w.WriteHeader(status)

	errResp := make(map[string]string)
	errResp["error"] = message
	jsonErr, err := json.Marshal(errResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(jsonErr)
}

func main() {
	// Read from .env when not in production
	// In production the env variables will already be set correctly
	if os.Getenv("APP_ENV") != "production" {
		envErr := godotenv.Load()
		if envErr != nil {
			log.Fatal("Error loading .env file")
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/upload", Upload)

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("No port set")
	}
	// TODO: Setup allowed origins and methods for cors to env variables
	handler := cors.Default().Handler(mux)
	err := http.ListenAndServe(":"+port, handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
