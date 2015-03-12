package lib

import (
	"io/ioutil"
	"net/http"
	"time"
	"net"
	"bytes"
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

	for headerName, headerValue := range r.RequestOptions.Headers {
		req.Header.Add(headerName, headerValue)
	}

	if (err != nil) {
		return ResponseStats {
			TimeToConnect: r.ConnectionTime,
			TimeToRespond: r.RequestTime,
			TotalTime: r.TotalTime,
			StartTime: startTime,
			FinishTime: time.Now(),
			Failures : []DescriptiveError{*NewRequestExecutionError(err)},
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
			Failures : []DescriptiveError{*NewRequestExecutionError(err)},
		}, err
	}
	finishTime := time.Now()

	r.TotalTime = time.Since(startTime)
	r.RequestTime = r.TotalTime - r.ConnectionTime

	reqBody, respBody, _ := r.isolatePayloads(req, resp)

	failures := []DescriptiveError{}

	respHeaderError := ValidateRespHeaders(r.RequestOptions.RespHeaders, resp)
	if (respHeaderError != nil) {
		failures = append(failures, respHeaderError)
	}

	statusFailure := ValidateStatusCode(r.RequestOptions.ResponseCode, resp)
	if (statusFailure != nil) {
		failures = append(failures, statusFailure)
	}

	if (r.RequestOptions.JSONSchema != ""){
		err = ValidateSchema(respBody, resp, r.RequestOptions.JSONSchema)
		if (err != nil) {
			descriptiveErr, _ := err.(DescriptiveError)
			failures = append(failures, descriptiveErr)
		}
	}

	return ResponseStats {
		TimeToConnect: r.ConnectionTime,
		TimeToRespond: r.RequestTime,
		TotalTime: r.TotalTime,
		StartTime: startTime,
		FinishTime: finishTime,

		Failures : failures,

		ReqPayload : reqBody,
		RespPayload : respBody,
	}, err
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
