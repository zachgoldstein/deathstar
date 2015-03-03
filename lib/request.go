package lib

import (
	"io/ioutil"
	"net/http"
	"time"
	"net"
	"bytes"
	"github.com/xeipuuv/gojsonschema"
	"fmt"
	"sort"
	"net/http/httputil"
)

type RequestRecorder struct {
	ConnectionTime time.Duration
	RequestTime time.Duration
	TotalTime time.Duration
	RequestOptions RequestOptions
	Client *http.Client
	Transport *http.Transport
}

func NewRequestRecorder (reqOpts RequestOptions) *RequestRecorder {
	recorder := &RequestRecorder{
		RequestOptions : reqOpts,
	}
	recorder.Client = recorder.createHttpClient()
	return recorder
}

func (r *RequestRecorder) PerformRequest() (respStats ResponseStats, err error){

	startTime := time.Now()

	req, err := r.constructRequest()

	if (r.RequestOptions.EnableKeepAlive) {
		req.Header.Add("Connection", "keep-alive")
	} else {
		req.Close = true
	}

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

	resp, err := r.issueRequest(req)
	if (err != nil) {
		req.Body.Close()
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
//	reqBody, respBody := "",""

	valid, validationErr, respErr, failCategory, err := r.validateResponse(respBody, resp, r.RequestOptions.JSONSchema)
//	valid, validationErr, respErr, failCategory, err := true, false, false, "", nil

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

func (r *RequestRecorder) createHttpClient() (*http.Client) {
	client := http.DefaultClient
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DisableKeepAlives : !r.RequestOptions.EnableKeepAlive,
		DisableCompression : true,
		MaxIdleConnsPerHost : 2,
		Dial: r.DialWithTimeRecorder,
		TLSHandshakeTimeout: r.RequestOptions.TLSHandshakeTimeout,
	}

	client.Timeout = r.RequestOptions.Timeout
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

	if (err != nil) {
		Log("temp", "dialer err? ",err)
		return conn, err
	}

	r.ConnectionTime = time.Since(now)

	return conn, err
}

func (r *RequestRecorder) issueRequest(req *http.Request)(resp *http.Response, err error) {
	return r.Client.Do(req)
}

func (r *RequestRecorder) isolatePayloads (req *http.Request, resp *http.Response) (reqPayload string, respPayload string, err error) {

	respDump, err := httputil.DumpResponse(resp, true)
	Log("debug", "DEBUGGING RAW RESPONSE ================== /n ",string(respDump), " /n ==================")

	respPayloadBytes, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	respPayload = string(respPayloadBytes)
	if (err != nil) {
		return reqPayload, respPayload, err
	}

	reqPayloadBytes, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
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
