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
	Done chan bool
	StatsChan chan ResponseStats
	RequestOptions RequestOptions
	Started bool
	Stopped bool

	Requester *RequestRecorder
	CustomClient *http.Client
}

func NewExecutor(id string, requestChan chan bool, statsChan chan ResponseStats, reqOpts RequestOptions) *Executor {
	newExecutor :=  &Executor{
		Done : make (chan bool),
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
	if (e.Stopped) { return }

	e.Started = true

	e.Requester = NewRequestRecorder(e.RequestOptions)
	if e.HasCustomClient() {
		e.Requester.Client = e.CustomClient
	}

	for j := range e.RequestChan {
		e.IsExecuting = true
		Log("execute", fmt.Sprintln("executor", e.Id, "issuing request", j) )
		stats, err := e.Requester.PerformRequest()
		if (err != nil) {
			Log( "all", fmt.Sprintln("An error occurred executing request, ", err) )
		}

		Log("execute", fmt.Sprintln("executor", e.Id, "returning stats", j) )
		e.IsExecuting = false
		e.StatsChan <- stats
		if (e.Stopped) {
			e.Done <- true
		}
	}
}

func (e *Executor) Stop() {
	e.Stopped = true
	if (e.IsExecuting) {
		for _ = range e.Done {
			return
		}
	}
	return
}

func testRequest() ResponseStats {
	duration := time.Millisecond * time.Duration(3000 * rand.Float64())
	Log("execute", fmt.Sprintln("test request length ",duration) )
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
