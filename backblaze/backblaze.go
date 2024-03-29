package backblaze

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
)

// TODO: These should be uppercase
type B2BackBlazeClient struct {
	authorizeUrl  string
	loginAuth     string
	getUploadPath string
	bucketId      string
	bucketName    string
}

type AuthResponse struct {
	ApiUrl             string `json:"apiUrl"`
	AuthorizationToken string `json:"authorizationToken"`
	DownloadUrl        string `json:"downloadUrl"`
}

type UploadUrlResponse struct {
	AuthorizationToken string `json:"authorizationToken"`
	BucketId           string `json:"bucketId"`
	UploadUrl          string `json:"uploadUrl"`
}

type UploadMeta struct {
	AccountId       string      `json:"accountId"`
	BucketId        string      `json:"bucketId"`
	ContentLength   int         `json:"contentLength"`
	ContentSha1     string      `json:"contentSha1"`
	ContentType     string      `json:"contentType"`
	FileId          string      `json:"fileId"`
	FileInfo        interface{} `json:"fileInfo"`
	FileName        string      `json:"fileName"`
	UploadTimeStamp int         `json:"uploadTimestamp"`
}

type UploadResponse struct {
	DownloadUrl string
	ApiResponse UploadMeta
}

type UploadFile struct {
	Bytes   []byte
	Handler *multipart.FileHeader
	Error   error
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
		authResp.ApiUrl+b2.getUploadPath,
		bytes.NewBuffer(jsonStr))
	// Set the auth token we received
	req.Header.Set("Authorization", authResp.AuthorizationToken)
	// THIS could be a sendRequest method
	client := &http.Client{}
	log.Println("Fetching an upload URL")
	resp, err := client.Do(req)
	if err != nil {
		log.Println("getUploadUrl: Request failed. ", err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("getUploadUrl: Couldn't read the body. ", err.Error())
	}

	var result UploadUrlResponse
	jsonErr := json.Unmarshal(body, &result)
	if jsonErr != nil {
		log.Println("getUploadUrl: Couldn't unmarshal?", jsonErr.Error())
	}

	return result
}

// Upload the file.
// We'll need URL and authorization data from the UploadUrlResponse,
// the bytes we're uploading, and the file name and size from the file header.
func (b2 B2BackBlazeClient) uploadFile(
	authResp AuthResponse,
	fileBytes []byte,
	handler *multipart.FileHeader) <-chan http.Response {

	requestChan := make(chan http.Response)
	go func() {
		log.Println("Uploading: ", handler.Filename)
		// TODO: Would be nice if we didnt have to get a new upload URL everytime?
		uploadUrlResp := b2.getUploadUrl(authResp)
		log.Println("Upload URL is: ", uploadUrlResp.UploadUrl)

		req, err := http.NewRequest(
			http.MethodPost,
			uploadUrlResp.UploadUrl,
			bytes.NewReader(fileBytes),
		)

		fileType := http.DetectContentType(fileBytes)
		checkSum := sha1CheckSumString(fileBytes)

		headers := map[string]string{
			"Authorization":     uploadUrlResp.AuthorizationToken,
			"X-Bz-File-Name":    handler.Filename,
			"Content-Type":      fileType,
			"Content-Length":    strconv.FormatInt(handler.Size, 10),
			"X-Bz-Content-Sha1": checkSum,
		}
		for header, v := range headers {
			req.Header.Set(header, v)
		}
		// SEND REQUEST METHOD
		client := &http.Client{}
		log.Println("Making upload request...")
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("uploadFile: Request failed. ", err.Error())
			resp.Body.Close()
		}
		requestChan <- *resp
		log.Println("Upload file response code: ", resp.Status)

		close(requestChan)
	}()
	return requestChan
}

// Creates the download URL for the uploaded file
func (b2 B2BackBlazeClient) makeDownloadUrl(authResp AuthResponse, fileName string) string {
	return authResp.DownloadUrl + "/file/" + b2.bucketName + "/" + fileName
}

// Make a simple HTTP request
// TODO: This doesn't really need to exist, only used one place
func makeHttpRequest(method string, url string, authToken string) (resp *http.Response, err error) {
	client := &http.Client{}
	req, _ := http.NewRequest(method, url, nil)
	req.Header.Set("Authorization", authToken)

	return client.Do(req)
}

// Create the SHA1 checksum of a file
func sha1CheckSumString(fileBytes []byte) string {
	hasher := sha1.New()
	hasher.Write(fileBytes)
	checkSum := hasher.Sum(nil)
	hashString := fmt.Sprintf("%x", checkSum)

	return hashString
}

// Create the b2 client for the upload
func MakeB2Client() B2BackBlazeClient {
	return B2BackBlazeClient{
		os.Getenv("B2_AUTHORIZE_URL"),
		os.Getenv("B2_LOGIN_AUTH"),
		os.Getenv("B2_GET_UPLOAD_PATH"),
		os.Getenv("B2_BUCKET_ID"),
		os.Getenv("B2_BUCKET_NAME"),
	}
}

// Save a file(s) to the backblaze cloud
func Save(payloads []UploadFile) (map[string]UploadResponse, error) {
	b2 := MakeB2Client()
	// TODO: Would be nice to not have to do this everytime if we dont need to
	authResp := b2.authorizeAccount()
	// Initialize a slice of http.Response channels
	var chans []<-chan http.Response
	// Intialize a slice of UploadResponse
	responses := make(map[string]UploadResponse)
	// Iterate over our request payloads and append the response channels into our channel slice.
	// This allows our requests to run concurrently while we're able to store the
	// response from each request before continuing
	for _, payload := range payloads {
		chans = append(chans, b2.uploadFile(authResp, payload.Bytes, payload.Handler))
	}
	// Iterate over our responses and prepare a map of UploadResponse structs
	for _, response := range chans {
		if resp := <-response; resp.StatusCode == http.StatusOK {
			// Create struct with the response and the download URL
			bodyBytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				// TODO: Don't want this... will exit the application
				log.Fatal(err)
			}
			uploadMeta := UploadMeta{}
			if err := json.Unmarshal(bodyBytes, &uploadMeta); err != nil {
				// TODO: Don't want this... will exit the application
				log.Fatal(err)
			}
			downloadUrl := b2.makeDownloadUrl(authResp, uploadMeta.FileName)
			log.Println(fmt.Sprintf("Download URL for %s is %s", uploadMeta.FileName, downloadUrl))

			responses[uploadMeta.FileName] = UploadResponse{downloadUrl, uploadMeta}
			// Something went wrong...
		} else {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			// Log it out for now since I'm not prepared for this
			log.Println(string(bodyBytes))
			err := errors.New(string(bodyBytes))
			// Exit function with error
			// TODO: Probably a better way to handle this though
			return responses, err
		}
	}
	// Everything went well... probably
	return responses, nil
}
