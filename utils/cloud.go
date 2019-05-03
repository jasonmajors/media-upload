package utils

import (
	"fmt"
	// "os"
	"strconv"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"bytes"
	// "mime/multipart"
	"crypto/sha1"
	// "encoding/base64"
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

// Request our APi information from our account ID and application key
// This will give us the API URL, the token for authenticating, and our download URL
func authorizeAccount(url string) AuthResponse {
	resp, err := makeHttpRequest(http.MethodGet, url, loginToken)
	if err != nil {
		fmt.Println("authorizeAccount: The request failed.")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("authorizeAccount: Couldn't read the body. ", err.Error())
	}

	var result AuthResponse
	jsonErr := json.Unmarshal(body, &result)
	if jsonErr != nil {
		fmt.Println("authorizeAccount: Couldn't unmarshal? ", jsonErr.Error())
	}

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
	// Make the JSON
	var jsonStr = []byte(fmt.Sprintf(`{"bucketId":"%s"}`, bucketId))
	// Build the POST request
	req, _ := http.NewRequest(
		http.MethodPost,
		authResp.ApiUrl + getUploadPath,
		bytes.NewBuffer(jsonStr))
	// Set the auth token we received
	req.Header.Set("Authorization", authResp.AuthorizationToken)
	// THIS could be a sendRequest method
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

	return result
}

func sha1CheckSumString(fileBytes []byte) string {
	hasher := sha1.New()
	hasher.Write(fileBytes)
	checkSum := hasher.Sum(nil)
	hashString := fmt.Sprintf("%x", checkSum)

	return hashString
}

func uploadFile(
	uploadUrlResp UploadUrlResponse,
	fileBytes []byte,
	fileName string,
	fileSize int64) {

	req, err := http.NewRequest(
		http.MethodPost,
		uploadUrlResp.UploadUrl,
		bytes.NewReader(fileBytes),
	)

	fileType := http.DetectContentType(fileBytes)
	checkSum := sha1CheckSumString(fileBytes)

	fmt.Println(fileType)
	fmt.Println(uploadUrlResp.AuthorizationToken)
	fmt.Println(fileName)
	fmt.Println(strconv.FormatInt(fileSize, 10))
	fmt.Println(checkSum)
	// I think this is wrong

	headers := map[string]string{
		"Authorization": uploadUrlResp.AuthorizationToken,
		"X-Bz-File-Name": fileName,
		"Content-Type": fileType,
		"Content-Length": strconv.FormatInt(fileSize, 10),
		// "X-Bz-Content-Sha1": base64.URLEncoding.EncodeToString(checkSum),
		"X-Bz-Content-Sha1": checkSum,
	}
	for header, v := range headers {
		req.Header.Set(header, v)
	}
	// SEND REQUEST METHOD
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("uploadFile: Request failed. ", err.Error())
	}
	defer resp.Body.Close()

	fmt.Println(resp.Status)
}

func Save(w http.ResponseWriter, fileBytes []byte, fileName string, fileSize int64) {
	authResp := authorizeAccount(authorizeUrl)
	uploadUrlResp := getUploadUrl(authResp)
	uploadFile(uploadUrlResp, fileBytes, fileName, fileSize)
}
