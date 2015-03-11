package lib

import (
	"github.com/xeipuuv/gojsonschema"
	"fmt"
	"sort"
	"net/http"
)

type DescriptiveError interface {
	error
	Description() string
	Category() string
}

type DisplayableError struct {
	DescriptiveError
	category string
}

type RequestExecutionError struct {
	DisplayableError
	err error
}

func (e RequestExecutionError) Error() string {
	return e.err.Error()
}

func (e RequestExecutionError) Description() string {
	return fmt.Sprintf("An error occurred executing the request %v", e.err.Error())
}

func (e RequestExecutionError) Category() string {
	return e.category
}

func NewRequestExecutionError(err error) *RequestExecutionError{
	return &RequestExecutionError{
		err : err,
		DisplayableError: DisplayableError{category : "RequestExecutionError",},
	}
}

type StatusCodeError struct {
	DisplayableError
	StatusCode int
}

func NewStatusCodeError(statusCode int) *StatusCodeError {
	return &StatusCodeError{
		StatusCode : statusCode,
		DisplayableError: DisplayableError{category : "StatusCode",},
	}
}

func (e StatusCodeError) Error() string {
	return fmt.Sprintf("Invalid status code returned, %v",e.StatusCode)
}

func (e StatusCodeError) Description() string {
	return "An invalid status code was returned"
}

func (e StatusCodeError) Category() string {
	return e.category
}


func ValidateStatusCode(expectedStatusCode int, resp *http.Response) (err DescriptiveError)  {
	if (resp.StatusCode != expectedStatusCode) {
		return *NewStatusCodeError(resp.StatusCode)
	}
	return nil
}

type ValidationError struct {
	DisplayableError
	Msg string
	errs []gojsonschema.ResultError
	errMsgs []string
}

func (e ValidationError) Error() string {
	errMsgs := []string{}
	for _, err := range e.errs {
		errMsgs = append(errMsgs, fmt.Sprintf("Validation Error: %v", err))
	}
	sort.Strings(errMsgs)
	e.errMsgs = errMsgs
	e.Msg = fmt.Sprint(errMsgs)
	return e.Msg
}

func (e ValidationError) Description() string {
	return "Validation against the schema failed."
}

func (e ValidationError) Category() string {
	return e.category
}


func NewValidationError (res *gojsonschema.Result) *ValidationError{
	return &ValidationError{
		errs : res.Errors(),
		DisplayableError: DisplayableError{category : "Validation",},
	}
}

func ValidateSchema(respPayload string, resp *http.Response, schema string) (err error) {
	responseLoader := gojsonschema.NewStringLoader(respPayload)
	schemaLoader := gojsonschema.NewStringLoader(schema)
	res, err := gojsonschema.Validate(schemaLoader, responseLoader)
	if (err != nil) {
		return *NewRequestExecutionError(err)
	}

	if !res.Valid() {
		return *NewValidationError(res)
	}

	return nil
}
