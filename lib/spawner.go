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
	OverallTicker *time.Ticker
	StartTime time.Time

	RequestChan chan bool
	StatsChan chan ResponseStats
	OverallStatsChan chan OverallStats
	Done chan bool

	Duration time.Duration
	RequestOptions RequestOptions
	Started bool
	CustomClient *http.Client
}

type ResponseStats struct {
	TimeToConnect time.Duration
	TimeToRespond time.Duration
	TotalTime time.Duration
	ResponsePayload []byte
	Failure bool
	FailCategory string
	ValidationErr bool
	RespErr bool
	ReqPayload string
	RespPayload string
}

type OverallStats struct {
	StartTime time.Time
	TotalTestDuration time.Duration
	TimeElapsed time.Duration

	NumExecutors int
	NumBusyExecutors int
	NumAvailableExecutors int
}

const tickerSecFrequency = 1
const overallStatsTickerFrequency = 100

func NewSpawner(rate int, maxExecutionTime time.Duration, responseStatsChan chan ResponseStats, overallStatsChan chan OverallStats, reqOpts RequestOptions) *Spawner {
	return &Spawner{
		Ticker : time.NewTicker(time.Second * tickerSecFrequency),
		OverallTicker : time.NewTicker(time.Millisecond * overallStatsTickerFrequency),
		Rate : rate,
		RequestChan : make(chan bool),
		Done : make(chan bool),
		StatsChan: responseStatsChan,
		OverallStatsChan: overallStatsChan,
		Duration: maxExecutionTime,
		RequestOptions : reqOpts,
	}
}

func (s *Spawner) Start () {
	Log("spawn", fmt.Sprintln("Spawning requests for ",s.Duration, " seconds") )

	Log("spawn", fmt.Sprintln("Blocking select for ticks and timeouts") )

	Log("spawn", fmt.Sprintln("timeout duration ",s.Duration) )
	timeoutTimer := time.NewTimer(s.Duration)
	s.StartTime = time.Now()

	//Goroutine to trigger periodic requests and stats
	go func () {
		for {
			select {
			case tick := <-s.Ticker.C:
				Log("spawn", fmt.Sprintln("TICK at ",tick) )
				s.MakeRequests()
			case _ = <-s.OverallTicker.C:
				s.SendOverallStats()
			case timeout := <-timeoutTimer.C:
				Log("spawn", fmt.Sprintln("Timed out, ",timeout) )
				s.SendOverallStats()
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

func (s *Spawner) SendOverallStats() {
	overallStats := OverallStats {
		NumExecutors : len(s.ExecutorPool),
		StartTime : s.StartTime,
	}

	for _, executor := range s.ExecutorPool {
		if !executor.IsExecuting {
			overallStats.NumAvailableExecutors += 1
		} else {
			overallStats.NumBusyExecutors += 1
		}
	}

	overallStats.TimeElapsed = time.Since(s.StartTime)
	overallStats.TotalTestDuration = s.Duration

	s.OverallStatsChan <- overallStats
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
		Log("spawn", fmt.Sprintln("Adding ",numToAdd," executors to pool") )

		for i:= 0; i < numToAdd; i++ {
			Log("spawn", fmt.Sprintln("Adding executor to pool", string(len(s.ExecutorPool) + i)) )

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

