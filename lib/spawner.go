package lib

import (
	"time"
	"fmt"
)

//Spawner is responsible for initiating requests on a channel at a specific rate
//It manages a pool of executors that will create and issue requests
type Spawner struct {
	Rate uint
	ExecutorPool []*Executor
	Ticker *time.Ticker
	RequestChan chan bool
	StatsChan chan ResponseStats
	Duration time.Duration
	Done chan bool
}

type ResponseStats struct {
	TimeToConnect time.Duration
	TimeToRespond time.Duration
}

const tickerSecFrequency = 1

func NewSpawner(rate uint, maxExecutionTime time.Duration, responseStatsChan chan ResponseStats) *Spawner {
	return &Spawner{
		Ticker : time.NewTicker(time.Second * tickerSecFrequency),
		Rate : rate,
		RequestChan : make(chan bool),
		Done : make(chan bool),
		StatsChan: responseStatsChan,
		Duration: maxExecutionTime,
	}
}

func (s *Spawner) Start () {
	fmt.Println("Spawning requests for ",s.Duration, " seconds")

	s.ExecutorPool = make([]*Executor, s.Rate)
	for i:= 0; i < int(s.Rate); i++ {
		executor := NewExecutor(fmt.Sprint(i), s.RequestChan, s.StatsChan)
		fmt.Println("Created executor ",fmt.Sprint(i))
		s.ExecutorPool[i] = executor
	}

	fmt.Println("Blocking select for ticks and timeouts")

	fmt.Println("timeout duration ",s.Duration)
	timeoutTimer := time.NewTimer(s.Duration)

	go func () {
		for {
			select {
			case tick := <-s.Ticker.C:
				fmt.Println("TICK at ",tick)
				s.InitiateRequests()
			case timeout := <-timeoutTimer.C:
				fmt.Println("Timed out, ",timeout)
				s.Ticker.Stop()
				//TODO: make sure that does not send done until all requests are finished
				// use another goroutine and check every request after the ticker is stopped
				s.Done <- true
			}
		}
	}()
}

func (s *Spawner) InitiateRequests() {
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
			newExecutor := NewExecutor(fmt.Sprint(len(s.ExecutorPool) + i), s.RequestChan, s.StatsChan)
			newExecutors = append(newExecutors, newExecutor)
		}
	}

	s.ExecutorPool = append(s.ExecutorPool, newExecutors...)

	for i:= 0; i < int(s.Rate); i++ {
		s.RequestChan <- true
	}
}
