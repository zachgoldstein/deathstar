package lib

import (
	"time"
	"flag"
	"runtime"
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
	"errors"
)

type RequestOptions struct {

	//Request control params
	URL string
	Method string
	Headers map[string]string
	Payload []byte
	Timeout time.Duration
	KeepAlive time.Duration
	EnableKeepAlive bool
	TLSHandshakeTimeout time.Duration

	//Execution control params
	Mode string

	Rate float64
	CPUs int
	RequestsToIssue int
	Concurrency int

	ExecuteSingleRequest bool
	IncreaseRateToFailure bool

	MaxExecutionSecs int
	MaxExecutionTime time.Duration

	WarmUpSecs int
	WarmUpTime time.Duration

	AnalaysisFreqMs int
	AnalaysisFreqTime time.Duration

	RenderFrequencyMs int
	RenderFrequency time.Duration

	//Failure detection params
	Harvest float64
	Yield float64
	Throughput float64
	PercentileLatencies []float64
	Percentiles []float64
	ResponseCode int

	//Validation params
	JSONSchema string
	RespHeaders map[string]string
}

type OutputOptions struct {
	ShowHTML bool
	ShowCLI bool
}

var DefaultRequestOptions RequestOptions = RequestOptions{
	URL : "http://localhost:8080/test/fail/validate",
	Method : "GET",
	ResponseCode : 200,

	Timeout : time.Second * 2,
	KeepAlive : time.Second * 2,
	EnableKeepAlive : false,
	TLSHandshakeTimeout : time.Second * 2,
	CPUs : runtime.NumCPU(),
	Rate : 10,
	Concurrency: 5,

	MaxExecutionSecs : 30*60,
	WarmUpSecs : 2,
	Percentiles : []float64{0.01, 0.05, 0.25, 0.50, 0.75, 0.95, 0.99, 0.999, 0.9999},

	AnalaysisFreqMs : 200,
	RenderFrequencyMs : 400,

	RequestsToIssue : 5000,

	Harvest : 85,
	Yield : 85,
	Throughput: 5,

	JSONSchema : "./lib/exampleSchema.json",
}

var DefaultOutputOptions OutputOptions = OutputOptions{
	ShowHTML : true,
	ShowCLI : true,
}

var DefaultMode = "scale"


// digestOptions will combine command line options and the config json file to create the options objects
func digestOptions()(reqOpts RequestOptions, outOpts OutputOptions, err error) {
	defaultReqOpts := DefaultRequestOptions
	defaultOutOpts := DefaultOutputOptions

	//Request control params
	url := flag.String("url", defaultReqOpts.URL , "the url to test")
	method := flag.String("method", defaultReqOpts.Method , "the url method to use")
	defaultHeaders := fmt.Sprintf("%v",defaultReqOpts.Headers)
	reqHeaderStr := flag.String("headers", defaultHeaders , "Requests headers for requests, in the form of a comma separated list; 'Max-Forwards:10,Accept-Charset:utf-8'")

	//Validation params
	jsonSchemaLocation := flag.String("schema", defaultReqOpts.JSONSchema, "The location of the schema file")

	defaultRespHeaders := fmt.Sprintf("%v",defaultReqOpts.RespHeaders)
	respHeaderStr := flag.String("respheaders", defaultRespHeaders, "Response headers to validate in responses, in the form of a comma separated list; 'Max-Forwards:10,Accept-Charset:utf-8'")

	//Execution control params
	showCLI := flag.Bool("cli", defaultOutOpts.ShowCLI, "show fancy cli")
	showHTML := flag.Bool("html", defaultOutOpts.ShowHTML, "serve fancy html")
	rate := flag.Float64("rate", defaultReqOpts.Rate, "req/s to issue")
	numReq := flag.Int("reqs", defaultReqOpts.RequestsToIssue, "Total requests to issue")
	concurrency := flag.Int("conc", defaultReqOpts.Concurrency, "Concurrent requests to issue")
	cpus := flag.Int("cpus", defaultReqOpts.CPUs, "CPUs to execute with")
	keepAlive := flag.Bool("keepalive", defaultReqOpts.EnableKeepAlive, "Execute with keep alive")

	executionSecs := flag.Int("time", defaultReqOpts.MaxExecutionSecs, "Maximum time (in secs) to execute the test")

	warmUpSecs := flag.Int("warmup", defaultReqOpts.WarmUpSecs, "Time until analysis starts")

	analysisFrequencyMs := flag.Int("analysis", defaultReqOpts.AnalaysisFreqMs, "Time in between each analysis run on the response data")
	renderFrequencyMs := flag.Int("render", defaultReqOpts.RenderFrequencyMs, "Time in between each push of data to the frontend")

	//Failure detection params
	expectedResponseCode := flag.Int("responsecode", defaultReqOpts.ResponseCode, "The expected response code for all requests")
	failureHarvest := flag.Float64("harvest", defaultReqOpts.Harvest, "The expected harvest % (percentage of requests that should get a response), below this value indicates a test failure")
	failureYield := flag.Float64("yield", defaultReqOpts.Yield, "The expected yield % (percentage of responses that should validate), below this value indicates a test failure")
	failureThroughput := flag.Float64("throughput", defaultReqOpts.Throughput, "The expected resp/s that should be returned by the test, below this value indicates a test failure")

	defaultPercentileLatencies := fmt.Sprintf("%v",defaultReqOpts.PercentileLatencies)
	failurePercentilesString := flag.String("percentiles", defaultPercentileLatencies , "The expected percentile latencies (in the form of a comma separated list) to achieve in the test, latencies below these values indicate a test failure. Latencies are for the 1, 5, 25, 50, 75, 95, 99, 99.9, 99.99 percentiles")

	mode := flag.String("mode", DefaultMode , "'fail' to continually ramp up request speed until failure, 'scale' for a test with consistent load, 'valid' for a test with a single request")
	Log("top", fmt.Sprintf("Starting in '%v' mode", *mode) )

	flag.Parse()

	reqHeaders, err := parseHeaders(*reqHeaderStr)
	if (err != nil) {
		return
	}

	respHeaders, err := parseHeaders(*respHeaderStr)
	if (err != nil) {
		return
	}

	jsonSchema, err := ioutil.ReadFile(*jsonSchemaLocation)
	if (err != nil) {
		return reqOpts, outOpts, errors.New(fmt.Sprintf("Could not load schema file at %v err: %v",*jsonSchemaLocation, err))
	}

	executionTime := time.Duration(*executionSecs) * time.Second
	warmUpTime := time.Duration(*warmUpSecs) * time.Second

	analysisFrequencyTime := time.Duration(*analysisFrequencyMs) * time.Millisecond
	renderFrequencyTime := time.Duration(*renderFrequencyMs) * time.Millisecond

	failurePercentiles, err := parsePercentiles(*failurePercentilesString, defaultReqOpts.Percentiles)
	if (err != nil) {
		return
	}

	if (*mode == "fail") {
		reqOpts.IncreaseRateToFailure = true
	} else if (*mode == "scale" ) {
	} else if (*mode == "valid" ) {
		reqOpts.ExecuteSingleRequest = true
	}

	if *showCLI {
		showLogs = false
	}

	return RequestOptions{
		//Request control params
		Method : *method,
		URL : *url,
		Headers : reqHeaders,

		//Validation params
		JSONSchema : string(jsonSchema),
		RespHeaders : respHeaders,

		//Execution control params
		Mode : *mode,
		Timeout : defaultReqOpts.Timeout,
		KeepAlive : defaultReqOpts.KeepAlive,
		EnableKeepAlive : *keepAlive,
		TLSHandshakeTimeout : defaultReqOpts.TLSHandshakeTimeout,

		Rate : *rate,
		CPUs : *cpus,
		Concurrency : *concurrency,
		RequestsToIssue : *numReq,

		MaxExecutionTime : executionTime,
		WarmUpTime : warmUpTime,
		AnalaysisFreqTime : analysisFrequencyTime,
		RenderFrequency : renderFrequencyTime,

		//Failure detection params
		ResponseCode: *expectedResponseCode,
		Harvest: *failureHarvest,
		Yield: *failureYield,
		Throughput: *failureThroughput,
		PercentileLatencies: failurePercentiles,
		Percentiles : defaultReqOpts.Percentiles,

	}, OutputOptions {
		ShowHTML : *showHTML,
		ShowCLI: *showCLI,
	}, nil
}

func parsePercentiles(rawPercentileLatency string, percentiles []float64) (percentileLatencies []float64, err error) {
	rawPercentileLatency = strings.TrimPrefix(rawPercentileLatency, "[")
	rawPercentileLatency = strings.TrimSuffix(rawPercentileLatency, "]")
	percArr := strings.Split(rawPercentileLatency, ",")
	for _, percStr := range percArr {
		if (percStr == "") { continue }
		perc, err := strconv.ParseFloat(percStr, 64)
		if err != nil {
			return percentileLatencies, err
		}
		percentileLatencies = append(percentileLatencies, perc)
	}
	return
}

func parseHeaders(headerStr string) (headers map[string]string, err error) {
	headers = make(map[string]string)
	headerStr = strings.TrimPrefix(headerStr, "map")
	headerStr = strings.TrimPrefix(headerStr, "[")
	headerStr = strings.TrimSuffix(headerStr, "]")
	headerStrs := strings.Split(headerStr, ",")
	for _, rawHeader := range headerStrs {
		if (rawHeader == "") { continue }
		header := strings.Split(rawHeader, ":")
		if (len(header) != 2) {
			return headers, errors.New(fmt.Sprintf("There was an error parsing header, %v",rawHeader))
		}
		headers[header[0]] = header[1]
	}
	return
}
