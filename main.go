package main

import (
	"fmt"
	// "io"
	"log"
	"net/http"
	// "os"
	"io/ioutil"
	// "mime/multipart"

	"github.com/jasonmajors/utils"
)

const maxUploadSize = 2 * 1024 * 1024 // 2mb

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
		fallthrough
	case "image/png":
		valid = true
		break
	default:
		valid = false
	}
	return valid
}

func Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Check file size isnt too big
		if ok := fileSizeIsOk(w, r); ok != true {
			fmt.Fprintf(w, "The file's too big man")
			return
		}
		// Get the file
		file, handler, err := r.FormFile("uploadFile")
		if err != nil {
			fmt.Println("Unable to read file", err)
			return
		}
		defer file.Close()
		// Read all of the contents of our file into a byte array
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Println(err)
		}
		// Get filetype
		fileType := http.DetectContentType(fileBytes)
		// Check valid mimetype
		if valid := validateFileType(fileType); valid != true {
			fmt.Println("Upload: Invalid file type")
			return
		}
		// Create a temp file in /tmp
		tempFile, err := ioutil.TempFile("./tmp", "upload-*.png")
		if err != nil {
			fmt.Println(err)
		}
		defer tempFile.Close()
		// Write this byte array to our temp file
		tempFile.Write(fileBytes)
		// We did it
		fmt.Fprintf(w, "Uploaded file\n")
		// TODO: Could do this before saving it to /tmp or something
		utils.Save(w, fileBytes, handler)
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
