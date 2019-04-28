package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"io/ioutil"
)

const maxUploadSize = 2 * 1024

func Upload(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	if r.Method == "POST" {
		// Check file size isnt too big. TODO Move into its own method. Can return nil
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			fmt.Println("File too big")
		}
		// get the file?
		// TODO: move into own method.. return file
		file, handler, err := r.FormFile("uploadFile")
		if err != nil {
			fmt.Println("Unable to read file")
			fmt.Println(err)
			return
		}
		defer file.Close()
		// Check mimetype
		// TODO: move into own file.. return bytes?
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Println("Unable to read file")
		}
		fileType := http.DetectContentType(fileBytes)
		switch fileType {
		case "image/jpeg", "image/jpg":
		case "image/png":
			break
		default:
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
