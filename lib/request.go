package lib

import (
	"io/ioutil"
	"net/http"
	"time"
	"net"
	"bytes"
)

type RequestRecorder struct {
	ConnectionTime time.Duration
	RequestTime time.Duration
	TotalTime time.Duration
	RequestOptions RequestOptions
	CustomClient *http.Client
}

func NewRequestRecorder (reqOpts RequestOptions) *RequestRecorder {
	return &RequestRecorder{
		RequestOptions : reqOpts,
	}
}

func (r *RequestRecorder) PerformRequest() ResponseStats {

	now := time.Now()

	req, err := r.constructRequest()
	if (err != nil) {
		issueError(err)
	}

	client := r.createHttpClient()
	if r.HasCustomClient() {
		client = r.CustomClient
	}

	respBody, err := r.issueRequest(req, client)
	if (err != nil) {
		issueError(err)
	}

	r.TotalTime = time.Since(now)

	r.RequestTime = r.TotalTime - r.ConnectionTime

	return ResponseStats {
		TimeToConnect: r.ConnectionTime,
		TimeToRespond: r.RequestTime,
		TotalTime: r.TotalTime,
		ResponsePayload: respBody,
	}
}

func (r *RequestRecorder) constructRequest() (req *http.Request, err error) {
	return http.NewRequest(r.RequestOptions.Method, r.RequestOptions.URL, bytes.NewReader(r.RequestOptions.Payload))
}

func (r *RequestRecorder) createHttpClient() *http.Client {
	client := http.DefaultClient
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: r.DialWithTimeRecorder,
		TLSHandshakeTimeout: r.RequestOptions.TLSHandshakeTimeout,
	}
	client.Transport = transport

	return client
}

func (r *RequestRecorder) DialWithTimeRecorder(network, address string) (conn net.Conn, err error) {
	dialer := &net.Dialer{
		Timeout:   r.RequestOptions.Timeout,
		KeepAlive: r.RequestOptions.KeepAlive,
	}

	now := time.Now()

	conn, err = dialer.Dial(network, address)

	r.ConnectionTime = time.Since(now)

	return conn, err
}

func (r *RequestRecorder) issueRequest(req *http.Request, client *http.Client)(resPayload []byte, err error) {
	resp, err := client.Do(req)
	if ( err != nil) {
		return resPayload, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (r *RequestRecorder) HasCustomClient() bool {
	if (r.CustomClient != nil && r.CustomClient.Transport != nil) {
		return true
	}
	return false
}
