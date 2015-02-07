package lib

import (
	"net/http"
	"time"
)

type Executor struct {
	Req http.Request
	Complete bool
	Connecting bool
	Responding bool
}

type RequestResponseResult struct {
	TimeToConnect time.Time
	TimeToRespond time.Time
}

func NewExecutor() *Executor {

}

func (a *Executor) CreateRequest() http.Request {

}

