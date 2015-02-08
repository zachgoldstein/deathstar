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
}

func NewExecutor(id string, requestChan chan bool, statsChan chan ResponseStats) *Executor {
	newExecutor :=  &Executor{
		Id : id,
		RequestChan : requestChan,
		StatsChan : statsChan,
	}
	go newExecutor.Start()

	return newExecutor
}

// Start will cause the executor to pull off the channel instructions to issue requests,
// It will only attempt to receive off the channel when it's done its request response cycle.
func (e *Executor) Start(){
	for j := range e.RequestChan {
		fmt.Println("executor", e.Id, "issuing request", j)
		e.IsExecuting = true
		duration := time.Millisecond * time.Duration(3000 * rand.Float64())
		fmt.Println("request length ",duration)
		time.Sleep(duration)
		stats := ResponseStats{
			TimeToConnect : time.Millisecond,
			TimeToRespond : time.Second,
		}
		fmt.Println("executor ", e.Id, " sending response stats \n", stats)
		e.IsExecuting = false
		e.StatsChan <- stats
	}
}

func (e *Executor) createRequest() http.Request {
	return http.Request{}
}

func (e *Executor) issueRequest() http.Response {
	return http.Response{}
}

