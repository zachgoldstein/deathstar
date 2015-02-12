package lib

import "time"

type RequestOptions struct {
	Method string
	URL string
	Headers map[string]string
	CPUs uint
	Timeout time.Duration
	KeepAlive time.Duration
	TLSHandshakeTimeout time.Duration
	Payload []byte
	JSONSchema string
}

var DefaultRequestOptions RequestOptions = RequestOptions{
	Timeout : time.Second * 30,
	KeepAlive : time.Second * 30,
	TLSHandshakeTimeout : time.Second * 10,
	CPUs : 1,
}

type OutputOptions struct {
	OutputHTML bool
	ShowFullJSON bool
	HTMLOutputLocation string
}

type Options struct {
	Rate uint
}

// digestOptions will combine command line options and the config json file to create the options objects
func digestOptions()(reqOpts RequestOptions, outOpts OutputOptions, err error) {
	return RequestOptions{
		Method : "GET",
		URL : "http://localhost:8080/test/fail/error",
		JSONSchema : testJSONSchema,
	}, OutputOptions {
		OutputHTML : true,
		ShowFullJSON : true,
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
