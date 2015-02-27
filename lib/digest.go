package lib

import (
	"time"
	"flag"
	"runtime"
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

	Rate int
	CPUs int
	RequestsToIssue int
	Concurrency int

	ReqLimitMode string //Limits req/s to either req/s or max total number of reqs

	MaxExecutionSecs int
	MaxExecutionTime time.Duration
}

type OutputOptions struct {
	ShowHTML bool
	ShowCLI bool
	ShowFullJSON bool
}

var DefaultRequestOptions RequestOptions = RequestOptions{
	Timeout : time.Second * 2,
	KeepAlive : time.Second * 2,
	EnableKeepAlive : false,
	TLSHandshakeTimeout : time.Second * 2,
	CPUs : runtime.NumCPU(),
	Concurrency: 150,

	ReqLimitMode : "total",
	MaxExecutionSecs : 60,

	RequestsToIssue : 500,
}

var DefaultOutputOptions OutputOptions = OutputOptions{
	ShowHTML : true,
	ShowCLI : false,
	ShowFullJSON : true,
}

// digestOptions will combine command line options and the config json file to create the options objects
func digestOptions()(reqOpts RequestOptions, outOpts OutputOptions, err error) {
	defaultReqOpts := DefaultRequestOptions
	defaultOutOpts := DefaultOutputOptions

	url := flag.String("url", "http://localhost:8080/test/success", "the url to test")
	showCLI := flag.Bool("cli", defaultOutOpts.ShowCLI, "show fancy cli")
	showHTML := flag.Bool("html", defaultOutOpts.ShowHTML, "serve fancy html")
	rate := flag.Int("rate", defaultReqOpts.Rate, "req/s to issue")
	numReq := flag.Int("reqs", defaultReqOpts.RequestsToIssue, "Total requests to issue")
	concurrency := flag.Int("conc", defaultReqOpts.Concurrency, "Concurrent requests to issue")
	cpus := flag.Int("cpus", defaultReqOpts.CPUs, "CPUs to execute with")
	keepAlive := flag.Bool("keepalive", defaultReqOpts.EnableKeepAlive, "Execute with keep alive")

	executionSecs := flag.Int("time", defaultReqOpts.MaxExecutionSecs, "Maximum time (in secs) to execute the test")
	executionTime := time.Duration(*executionSecs) * time.Second

	mode := flag.String("mode", defaultReqOpts.ReqLimitMode, "Mode that decides how to limit reqs being issued. 'total' issues at max speed until all have been issued, 'rate' will issue at a specified rate until the max execution time occurs")

	flag.Parse()

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

		ReqLimitMode : *mode,
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
