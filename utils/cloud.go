package utils

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"bytes"
)

const authorizeUrl = "https://api.backblazeb2.com/b2api/v2/b2_authorize_account"
// TODO: This shouldnt be here
// Store account Id and application key in env and generate the token via base64 encode
const loginToken = "Basic YjE2YjYyN2Q2MDBmOjAwMTk5MjQ1NTgxODEyMmJiNTQ3MTY2MWRkZjE3OGFmNGU5ZTljMDNkOQ=="
const getUploadPath = "/b2api/v2/b2_get_upload_url"
const bucketId = "eb6196bb4672771d66a0001f"

type SaveToCloudStorage interface {
	Save()
}

type CloudStorage struct {
	service string
}

func (client CloudStorage) SaveFile() {

}

type AuthResponse struct {
	ApiUrl string `json:"apiUrl"`
	AuthorizationToken string `json:"authorizationToken"`
	DownloadUrl string `json:"downlloadUrl"`
}

type UploadUrlResponse struct {
	AuthorizationToken string `json:"authorizationToken"`
	BucketId string `json:"bucketId"`
	UploadUrl string `json:"uploadUrl"`
}

func makeHttpRequest(method string, url string, authToken string) (resp *http.Response, err error) {
	client := &http.Client{}
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("Authorization", authToken)

	return client.Do(req)
}

func authorizeAccount(url string) AuthResponse {
	resp, err := makeHttpRequest(http.MethodGet, url, loginToken)
	if err != nil {
		fmt.Println("authorizeAccount: The request failed.")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("authorizeAccount: couldnt read the body. ", err.Error())
	}

	var result AuthResponse
	jsonErr := json.Unmarshal(body, &result)
	if jsonErr != nil {
		fmt.Println("authorizeAccount: couldnt unmarshal? ", jsonErr.Error())
	}
	fmt.Println(result.ApiUrl)

	return result
	// Make a json response from our Foo struct
	// jsonResp, err := json.Marshal(result)
	// if err != nil {
	// 	fmt.Println("authorizeAccount: couldnt marshal? ", err.Error())
	// }
	// w.Header().Set("content-type", "application/json")
	// w.Write(jsonResp)
}

func getUploadUrl(authResp AuthResponse) UploadUrlResponse {
	var jsonStr = []byte(`{"bucketId":"eb6196bb4672771d66a0001f"}`)
	// Build the POST request
	req, _ := http.NewRequest(
		http.MethodPost,
		authResp.ApiUrl + getUploadPath,
		bytes.NewBuffer(jsonStr))
	// Set the auth token we received
	req.Header.Set("Authorization", authResp.AuthorizationToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("getUploadUrl: Request failed. ", err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("getUploadUrl: Couldn't read the body. ", err.Error())
	}

	var result UploadUrlResponse
	jsonErr := json.Unmarshal(body, &result)
	if jsonErr != nil {
		fmt.Println("getUploadUrl: Couldn't unmarshal?", jsonErr.Error())
	}
	fmt.Println(result.AuthorizationToken)

	return result
}

func Save(w http.ResponseWriter) {
	authResp := authorizeAccount(authorizeUrl)
	getUploadUrl(authResp)
}
