package utils

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

const authorizeUrl = "https://api.backblazeb2.com/b2api/v2/b2_authorize_account"
// TODO: This shouldnt be here
const loginToken = "Basic YjE2YjYyN2Q2MDBmOjAwMTk5MjQ1NTgxODEyMmJiNTQ3MTY2MWRkZjE3OGFmNGU5ZTljMDNkOQ=="

type SaveToCloudStorage interface {
	SaveFile()
}

type CloudStorage struct {
	service string
}

func (client CloudStorage) SaveFile() {

}

type Foo struct {
	Account string `json:"accountId"`
}

func requestToken(url string, w http.ResponseWriter) {
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", loginToken)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("requestToken: The request failed.")
	}
	fmt.Printf("Response: HTTP: %s\n", resp.Status)

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("requestToken: couldnt read the body. ", err.Error())
	}
	var result Foo
	jsonErr := json.Unmarshal(body, &result)
	if jsonErr != nil {
		fmt.Println("requestToken: couldnt unmarshal? ", jsonErr.Error())
	}
	jsonResp, err := json.Marshal(result)
	if err != nil {
		fmt.Println("requestToken: couldnt marshal? ", err.Error())
	}
	w.Header().Set("content-type", "application/json")
	w.Write(jsonResp)
}


func Save(w http.ResponseWriter) {
	requestToken(authorizeUrl, w)
}
