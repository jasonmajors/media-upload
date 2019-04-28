package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"io/ioutil"
	"mime/multipart"
)

const maxUploadSize = 2 * 1024 * 1024 // 2mb

func fileSizeIsOk(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		return false
	}
	return true
}

func validateFileType(file multipart.File) bool {
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Unable to read file")
	}
	fileType := http.DetectContentType(fileBytes)
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
	fmt.Println("method:", r.Method)
	if r.Method == "POST" {
		// Check file size isnt too big. TODO Move into its own method. Can return nil
		if ok := fileSizeIsOk(w, r); ok != true {
			fmt.Fprintf(w, "The file's too big man")
		}
		// TODO: move into own method.. return file
		file, handler, err := r.FormFile("uploadFile")
		if err != nil {
			fmt.Println("Unable to read file")
			fmt.Println(err)
			return
		}
		defer file.Close()
		if valid := validateFileType(file); valid != true {
			fmt.Println("Invalid file type asshole")
			return
		}
		// whatever
		fmt.Fprintf(w, "%v", handler.Header)
		// ?? This saves the file?
		f, err := os.OpenFile("./tmp/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		if err == nil {
			// WOOOOOOOO
			fmt.Println("WE FUCKING DID IT WE'RE A GO DEV")
		} else {
			// Shit
			fmt.Println("uh oh")
			fmt.Println(err)
			return
		}
		defer f.Close()
		// Saving the file to the filepath? Seems to work without this...
		io.Copy(f, file)
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
