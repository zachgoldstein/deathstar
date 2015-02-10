package lib

import (
	"time"
	"fmt"
	"net/http"
)

//Spawner is responsible for initiating requests on a channel at a specific rate
//It manages a pool of executors that will create and issue requests
type Spawner struct {
	Rate int
	ExecutorPool []*Executor
	Ticker *time.Ticker
	RequestChan chan bool
	StatsChan chan ResponseStats
	Duration time.Duration
	Done chan bool
	RequestOptions RequestOptions
	Started bool
	CustomClient *http.Client
}

type ResponseStats struct {
	TimeToConnect time.Duration
	TimeToRespond time.Duration
	TotalTime time.Duration
	ResponsePayload []byte
	NumExecutors int
}

const tickerSecFrequency = 1

func NewSpawner(rate int, maxExecutionTime time.Duration, responseStatsChan chan ResponseStats, reqOpts RequestOptions) *Spawner {
	return &Spawner{
		Ticker : time.NewTicker(time.Second * tickerSecFrequency),
		Rate : rate,
		RequestChan : make(chan bool),
		Done : make(chan bool),
		StatsChan: responseStatsChan,
		Duration: maxExecutionTime,
		RequestOptions : reqOpts,
	}
}

func (s *Spawner) Start () {
	fmt.Println("Spawning requests for ",s.Duration, " seconds")

	fmt.Println("Blocking select for ticks and timeouts")

	fmt.Println("timeout duration ",s.Duration)
	timeoutTimer := time.NewTimer(s.Duration)

	//Goroutine to attach the current concurrency to stats
	go func() {
		for newStats := range s.StatsChan{
			newStats.NumExecutors = len(s.ExecutorPool)
			s.StatsChan <-newStats
		}
	}()

	//Goroutine to trigger periodic requests
	go func () {
		for {
			select {
			case tick := <-s.Ticker.C:
				fmt.Println("TICK at ",tick)
				s.MakeRequests()
			case timeout := <-timeoutTimer.C:
				fmt.Println("Timed out, ",timeout)
				s.Ticker.Stop()
				//TODO: make sure that does not send done until all requests are finished
				// use another goroutine and check every request after the ticker is stopped
				s.Done <- true
			}
		}
	}()

	//Start any executors in the pool
	for _, executor := range s.ExecutorPool {
		if !executor.Started {
			go executor.Start()
		}
	}

	s.Started = true
}

func (s *Spawner) MakeRequests() {
	//Issue requests on the channel
	//If all of the executors are busy, expand the pool as neccessary and create more executors.
	newExecutors := []*Executor{}
	numAvailableExecutors := 0
	for _, executor := range s.ExecutorPool {
		if !executor.IsExecuting {
			numAvailableExecutors += 1
		}
	}

	if numAvailableExecutors < int(s.Rate) {
		numToAdd := int(s.Rate) - numAvailableExecutors
		fmt.Println("Adding ",numToAdd," executors to pool")
		for i:= 0; i < numToAdd; i++ {
			fmt.Println("Adding executor to pool", string(len(s.ExecutorPool) + i))
			newExecutor := NewExecutor(fmt.Sprint(len(s.ExecutorPool) + i), s.RequestChan, s.StatsChan, s.RequestOptions)
			if s.HasCustomClient() {
				newExecutor.CustomClient = s.CustomClient
			}
			newExecutors = append(newExecutors, newExecutor)
		}
	}

	s.ExecutorPool = append(s.ExecutorPool, newExecutors...)

	//Start executors if the spawner is started
	if s.Started {
		for _, executor := range s.ExecutorPool {
			if !executor.Started {
				go executor.Start()
			}
		}
	}

	for i:= 0; i < int(s.Rate); i++ {
		s.RequestChan <- true
	}
}

func (s *Spawner) HasCustomClient() bool {
	if (s.CustomClient != nil && s.CustomClient.Transport != nil) {
		return true
	}
	return false
}

