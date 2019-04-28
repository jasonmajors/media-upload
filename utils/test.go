package utils

import (
	"fmt"
	"net/http"
)

func Uploadd(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	if r.Method == "POST" {
		value := r.FormValue("key")
		fmt.Fprintf(w, "here it is")
		fmt.Fprintf(w, value)
	} else {
		fmt.Fprintf(w, "Invalid HTTP method")
	}
}
