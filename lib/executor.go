package lib

import (
	"net/http"
	"time"
	"fmt"
	"math/rand"
)

type Executor struct {
	Id string
	Req http.Request
	IsExecuting bool
	Connecting bool
	Responding bool
	RequestChan chan bool
	StatsChan chan ResponseStats
	RequestOptions RequestOptions
	Started bool
	CustomClient *http.Client
}

func NewExecutor(id string, requestChan chan bool, statsChan chan ResponseStats, reqOpts RequestOptions) *Executor {
	newExecutor :=  &Executor{
		Id : id,
		RequestChan : requestChan,
		StatsChan : statsChan,
		RequestOptions: reqOpts,
	}

	return newExecutor
}

// Start will cause the executor to pull off the channel instructions to issue requests,
// It will only attempt to receive off the channel when it's done its request response cycle.
func (e *Executor) Start(){
	e.Started = true
	for j := range e.RequestChan {
		e.IsExecuting = true
		fmt.Println("executor", e.Id, "issuing request", j)

		requester := NewRequestRecorder(e.RequestOptions)
		if e.HasCustomClient() {
			requester.CustomClient = e.CustomClient
		}
		stats := requester.PerformRequest()

		fmt.Println("executor", e.Id, "returning stats", j)
		e.IsExecuting = false
		e.StatsChan <- stats
	}
}

func testRequest() ResponseStats {
	duration := time.Millisecond * time.Duration(3000 * rand.Float64())
	fmt.Println("test request length ",duration)
	time.Sleep(duration)
	return ResponseStats{
		TimeToConnect : time.Millisecond,
		TimeToRespond : time.Second,
	}
}

func (e *Executor) HasCustomClient() bool {
	if (e.CustomClient != nil && e.CustomClient.Transport != nil) {
		return true
	}
	return false
}
