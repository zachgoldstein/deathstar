package lib

import (
	"time"
	"flag"
	"runtime"
	"fmt"
)

type RequestOptions struct {
	URL string
	Method string
	Headers map[string]string
	Timeout time.Duration
	KeepAlive time.Duration
	EnableKeepAlive bool
	TLSHandshakeTimeout time.Duration

	Payload []byte
	JSONSchema string

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

}

type OutputOptions struct {
	ShowHTML bool
	ShowCLI bool
}

var DefaultRequestOptions RequestOptions = RequestOptions{
	Timeout : time.Second * 2,
	KeepAlive : time.Second * 2,
	EnableKeepAlive : false,
	TLSHandshakeTimeout : time.Second * 2,
	CPUs : runtime.NumCPU(),
	Rate : 10,
	Concurrency: 5,

	MaxExecutionSecs : 5,
	WarmUpSecs : 2,

	AnalaysisFreqMs : 200,

	RequestsToIssue : 5000,
}

var DefaultOutputOptions OutputOptions = OutputOptions{
	ShowHTML : true,
	ShowCLI : false,
}

var DefaultMode = "scale"

// digestOptions will combine command line options and the config json file to create the options objects
func digestOptions()(reqOpts RequestOptions, outOpts OutputOptions, err error) {
	defaultReqOpts := DefaultRequestOptions
	defaultOutOpts := DefaultOutputOptions

	url := flag.String("url", "http://localhost:8080/test/success", "the url to test")
	showCLI := flag.Bool("cli", defaultOutOpts.ShowCLI, "show fancy cli")
	showHTML := flag.Bool("html", defaultOutOpts.ShowHTML, "serve fancy html")
	rate := flag.Float64("rate", defaultReqOpts.Rate, "req/s to issue")
	numReq := flag.Int("reqs", defaultReqOpts.RequestsToIssue, "Total requests to issue")
	concurrency := flag.Int("conc", defaultReqOpts.Concurrency, "Concurrent requests to issue")
	cpus := flag.Int("cpus", defaultReqOpts.CPUs, "CPUs to execute with")
	keepAlive := flag.Bool("keepalive", defaultReqOpts.EnableKeepAlive, "Execute with keep alive")

	executionSecs := flag.Int("time", defaultReqOpts.MaxExecutionSecs, "Maximum time (in secs) to execute the test")
	executionTime := time.Duration(*executionSecs) * time.Second

	warmUpSecs := flag.Int("warmup", defaultReqOpts.WarmUpSecs, "Time until analysis starts")
	warmUpTime := time.Duration(*warmUpSecs) * time.Second

	analysisFrequencyMs := flag.Int("analysis", defaultReqOpts.AnalaysisFreqMs, "Time in between each analysis run on the response data")
	analysisFrequencyTime := time.Duration(*analysisFrequencyMs) * time.Millisecond

	mode := flag.String("mode", DefaultMode , "'fail' to continually ramp up request speed until failure, 'scale' for a test with consistent load, 'valid' for a test with a single request")
	Log("top", fmt.Sprintf("Starting in '%v' mode", mode) )

	if (*mode == "fail") {
		reqOpts.IncreaseRateToFailure = true
	} else if (*mode == "scale" ) {
	} else if (*mode == "valid" ) {
		reqOpts.ExecuteSingleRequest = true
	}

	flag.Parse()

	if *showCLI {
		showLogs = false
	}

	return RequestOptions{
		Timeout : defaultReqOpts.Timeout,
		KeepAlive : defaultReqOpts.KeepAlive,
		EnableKeepAlive : *keepAlive,
		TLSHandshakeTimeout : defaultReqOpts.TLSHandshakeTimeout,

		Method : "GET",
		URL : *url,
		JSONSchema : testJSONSchema,

		Rate : *rate,
		CPUs : *cpus,
		Concurrency : *concurrency,
		RequestsToIssue : *numReq,

		MaxExecutionTime : executionTime,
		WarmUpTime : warmUpTime,
		AnalaysisFreqTime : analysisFrequencyTime,
	}, OutputOptions {
		ShowHTML : *showHTML,
		ShowCLI: *showCLI,
	}, nil
}

var testJSONSchema = `
{
    "$schema": "http://json-schema.org/draft-04/schema#",
    "title": "Product",
    "description": "A product from Acme's catalog",
    "type": "object",
    "properties": {
        "id": {
            "description": "The unique identifier for a product",
            "type": "integer"
        },
        "name": {
            "description": "Name of the product",
            "type": "string"
        },
        "stringNumber": {
            "description": "A number from 0-9",
            "type": "string",
            "pattern":"[0-9]"
        },
        "price": {
            "type": "number",
            "minimum": 0,
            "exclusiveMinimum": true
        },
        "tags": {
            "type": "array",
            "items": {
                "type": "string"
            },
            "minItems": 1,
            "uniqueItems": true
        }
    },
    "required": ["id", "name", "price"]
}
`
