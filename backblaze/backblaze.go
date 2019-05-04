package backblaze

import (
	"os"
	"log"
	"fmt"
	"strconv"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"bytes"
	"mime/multipart"
	"crypto/sha1"

	"github.com/joho/godotenv"
)
// TODO: These should be uppercase
type B2BackBlazeClient struct {
    authorizeUrl string
    loginAuth string
    getUploadPath string
	bucketId string
	bucketName string
}

type AuthResponse struct {
	ApiUrl string `json:"apiUrl"`
	AuthorizationToken string `json:"authorizationToken"`
	DownloadUrl string `json:"downloadUrl"`
}

type UploadUrlResponse struct {
	AuthorizationToken string `json:"authorizationToken"`
	BucketId string `json:"bucketId"`
	UploadUrl string `json:"uploadUrl"`
}

type UploadedResponse struct {
	DownloadUrl string `json:"downloadUrl,omitempty"`
}

// Request our APi information from our account ID and application key
// This will give us the API URL, the token for authenticating, and our download URL
func (b2 B2BackBlazeClient) authorizeAccount() AuthResponse {
	resp, err := makeHttpRequest(http.MethodGet, b2.authorizeUrl, b2.loginAuth)
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
}

// Request a URL to upload to with our authorization info
func (b2 B2BackBlazeClient) getUploadUrl(authResp AuthResponse) UploadUrlResponse {
	// Make the JSON
	var jsonStr = []byte(fmt.Sprintf(`{"bucketId":"%s"}`, b2.bucketId))
	// Build the POST request
	req, _ := http.NewRequest(
		http.MethodPost,
		authResp.ApiUrl + b2.getUploadPath,
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

// Upload the file.
// We'll need URL and authorization data from the UploadUrlResponse,
// the bytes we're uploading, and the file name and size from the file header.
func (b2 B2BackBlazeClient) uploadFile(
    uploadUrlResp UploadUrlResponse,
    fileBytes []byte,
    handler *multipart.FileHeader) bool {

	req, err := http.NewRequest(
		http.MethodPost,
		uploadUrlResp.UploadUrl,
		bytes.NewReader(fileBytes),
	)

	fileType := http.DetectContentType(fileBytes)
	checkSum := sha1CheckSumString(fileBytes)

	headers := map[string]string{
		"Authorization": uploadUrlResp.AuthorizationToken,
		"X-Bz-File-Name": handler.Filename,
		"Content-Type": fileType,
		"Content-Length": strconv.FormatInt(handler.Size, 10),
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
	// TODO: Check if 200
	fmt.Println(resp.Status)
	// If 200!
	if true {
		return true
	}
	return false
}

func makeHttpRequest(method string, url string, authToken string) (resp *http.Response, err error) {
	client := &http.Client{}
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("Authorization", authToken)

	return client.Do(req)
}

func sha1CheckSumString(fileBytes []byte) string {
	hasher := sha1.New()
	hasher.Write(fileBytes)
	checkSum := hasher.Sum(nil)
	hashString := fmt.Sprintf("%x", checkSum)

	return hashString
}

func Save(w http.ResponseWriter, fileBytes []byte, handler *multipart.FileHeader) {
	// Set env variables from our .env file
	err := godotenv.Load()
	if err != nil {
	  log.Fatal("Error loading .env file")
	}
	b2 := B2BackBlazeClient{
		os.Getenv("B2_AUTHORIZE_URL"), 
		os.Getenv("B2_LOGIN_AUTH"), 
		os.Getenv("B2_GET_UPLOAD_PATH"), 
		os.Getenv("B2_BUCKET_ID"),
		os.Getenv("B2_BUCKET_NAME"),
	}
	authResp := b2.authorizeAccount()
	uploadUrlResp := b2.getUploadUrl(authResp)
	uploaded := b2.uploadFile(uploadUrlResp, fileBytes, handler)
	
	if uploaded {
		// TODO: Dynamically load bucket name; could live in .env or fetch it from BUCKET_ID
		downloadUrl := authResp.DownloadUrl + "/file/" + b2.bucketName + "/" + handler.Filename
		uploadedResp := UploadedResponse{downloadUrl}
		js, err := json.Marshal(uploadedResp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		w.Write(js)
	}
}