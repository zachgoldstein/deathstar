package lib

import (
	"io/ioutil"
	"net/http"
	"time"
	"net"
	"bytes"
	"github.com/xeipuuv/gojsonschema"
	"fmt"
	"net/http/httputil"
	"sort"
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

func (r *RequestRecorder) PerformRequest() (respStats ResponseStats, err error){

	startTime := time.Now()

	req, err := r.constructRequest()
	req.Close = true
	if (err != nil) {
		return ResponseStats {
			TimeToConnect: r.ConnectionTime,
			TimeToRespond: r.RequestTime,
			TotalTime: r.TotalTime,
			StartTime: startTime,
			FinishTime: time.Now(),
			Failure : true,
			ValidationErr : false,
			RespErr : true,
			FailCategory : err.Error(),
		}, err
	}

	client := r.createHttpClient()
	if r.HasCustomClient() {
		client = r.CustomClient
	}

	resp, err := r.issueRequest(req, client)
	if (err != nil) {
		return ResponseStats {
			TimeToConnect: r.ConnectionTime,
			TimeToRespond: r.RequestTime,
			TotalTime: r.TotalTime,
			StartTime: startTime,
			FinishTime: time.Now(),
			Failure : true,
			ValidationErr : false,
			RespErr : true,
			FailCategory : err.Error(),
		}, err
	}
	finishTime := time.Now()

	r.TotalTime = time.Since(startTime)

	r.RequestTime = r.TotalTime - r.ConnectionTime

	reqBody, respBody, err := r.isolatePayloads(req, resp)

	valid, validationErr, respErr, failCategory, err := r.validateResponse(respBody, resp, r.RequestOptions.JSONSchema)

	return ResponseStats {
		TimeToConnect: r.ConnectionTime,
		TimeToRespond: r.RequestTime,
		TotalTime: r.TotalTime,
		StartTime: startTime,
		FinishTime: finishTime,
		Failure : !valid,
		ValidationErr : validationErr,
		RespErr : respErr,
		FailCategory : failCategory,
		ReqPayload : reqBody,
		RespPayload : respBody,
	}, nil
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

func (r *RequestRecorder) issueRequest(req *http.Request, client *http.Client)(resp *http.Response, err error) {
	return client.Do(req)
}

func (r *RequestRecorder) HasCustomClient() bool {
	if (r.CustomClient != nil && r.CustomClient.Transport != nil) {
		return true
	}
	return false
}

func (r *RequestRecorder) isolatePayloads (req *http.Request, resp *http.Response) (reqPayload string, respPayload string, err error) {

	respDump, err := httputil.DumpResponse(resp, true)
	Log("debug", "DEBUGGING RAW RESPONSE ================== /n ",string(respDump), " /n ==================")

	defer resp.Body.Close()
	respPayloadBytes, err := ioutil.ReadAll(resp.Body)
	respPayload = string(respPayloadBytes)
	if (err != nil) {
		return reqPayload, respPayload, err
	}

	defer req.Body.Close()
	reqPayloadBytes, err := ioutil.ReadAll(req.Body)
	reqPayload = string(reqPayloadBytes)
	if (err != nil) {
		return reqPayload, respPayload, err
	}

	return string(reqPayload), string(respPayload), err
}

func (r *RequestRecorder) validateResponse(respPayload string, resp *http.Response, schema string) (valid bool, validationErr bool, respErr bool, failCategory string, err error) {
	if resp.StatusCode != 200 {
		return false, false, true, resp.Status, nil
	}

	responseLoader := gojsonschema.NewStringLoader(respPayload)
	schemaLoader := gojsonschema.NewStringLoader(schema)
	res, err := gojsonschema.Validate(schemaLoader, responseLoader)
	if (err != nil) {
		return false, true, false, err.Error(), err
	}

	Log("debug", "VALID RESPONSE? ",res)

	if !res.Valid() {
		errors := []string{}
		for _, validateError := range res.Errors() {
			errors = append(errors, fmt.Sprintf("Validation Error: %v", validateError))
		}
		sort.Strings(errors)

		return false, true, false, fmt.Sprint(errors), nil
	}

	return true, false, false, "", nil
}
