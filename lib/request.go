package lib

import (
	"io/ioutil"
	"net/http"
	"strings"
)

func constructRequest(opts RequestOptions) (req *http.Request, err error) {
	method := "GET"
	reqURL := "http://localhost:8080/test"
	payload := ``
	return http.NewRequest(method, reqURL, strings.NewReader(payload))
}

func createHttpClient() *http.Client {
	return http.DefaultClient
}

func issueRequest(req *http.Request, client *http.Client)(resPayload []byte, err error) {
	resp, err := client.Do(req)
	if ( err != nil) {
		return resPayload, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
