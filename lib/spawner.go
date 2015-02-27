package lib

import (
	"time"
	"fmt"
	"net/http"
	"sync"
	"runtime"
)

//Spawner is responsible for initiating requests on a channel at a specific rate
//It manages a pool of executors that will create and issue requests
type Spawner struct {
	Rate int
	MaxExecutionTime time.Duration
	RequestOptions RequestOptions
	RequestsToIssue int
	ReqLimitMode string
	Concurrency int

	ExecutorPool []*Executor
	Ticker *time.Ticker
	OverallTicker *time.Ticker
	TimeoutTimer *time.Timer
	StartTime time.Time

	RequestChan chan bool
	StatsChan chan ResponseStats
	OverallStatsChan chan OverallStats
	Done chan bool

	Started bool
	CustomClient *http.Client

	mu sync.Mutex
	RequestsIssued int
}

type ResponseStats struct {
	StartTime time.Time
	FinishTime time.Time

	TimeToConnect time.Duration
	TimeToRespond time.Duration
	TotalTime time.Duration
	Failure bool
	FailCategory string
	ValidationErr bool
	RespErr bool
	ReqPayload string
	RespPayload string
}

type OverallStats struct {
	Rate int

	StartTime time.Time
	TotalTestDuration time.Duration
	TimeElapsed time.Duration

	RequestsIssued int

	NumExecutors int
	NumBusyExecutors int
	NumAvailableExecutors int
}

const tickerSecFrequency = 1
const overallStatsTickerFrequency = 100

func NewSpawner(responseStatsChan chan ResponseStats, overallStatsChan chan OverallStats, reqOpts RequestOptions) *Spawner {
	return &Spawner{
		RequestChan : make(chan bool),
		Done : make(chan bool),
		StatsChan: responseStatsChan,
		OverallStatsChan: overallStatsChan,

		Rate : reqOpts.Rate,
		MaxExecutionTime: reqOpts.MaxExecutionTime,
		RequestsToIssue : reqOpts.RequestsToIssue,
		RequestOptions : reqOpts,
		ReqLimitMode : reqOpts.ReqLimitMode,
		Concurrency : reqOpts.Concurrency,
	}
}

func (s *Spawner) Start () {

	Log("spawn", fmt.Sprintln("Spawner starting") )

	s.StartTime = time.Now()

	runtime.GOMAXPROCS(s.RequestOptions.CPUs)

	//TODO: executor pool IS concurrency. Size should be user tunable
	//reqs should NOT be triggered on a ticker, executors should pull them off the chan as soon as they can...
	//pool size should NOT change
	//scale tests test MAX rate given X total requests and Y concurrecy (pool size)
	//fail tests should ramp number of request/s by increasing the amount of reqs queued on chan periodically

	s.SetupExecutorPool()

	s.SetupOverallStatsPipe()

	s.SetupTimeout()

	s.StartRequests()

	s.Started = true

	Log("spawn", fmt.Sprintln("Spawner started ") )
}

func (s *Spawner) SetupTimeout() {
	s.TimeoutTimer = time.NewTimer(s.MaxExecutionTime)
	Log("spawn", fmt.Sprintln("Timeout timer has started, and will trigger in ",s.MaxExecutionTime) )
	go func () {
		for _ = range s.TimeoutTimer.C {
			Log("spawn", fmt.Sprintln("Timed out, ",time.Now()) )
			s.Stop()
		}
	}()
}

func (s *Spawner) Cleanup() {
	s.SendOverallStats()
}

func (s *Spawner) Stop() {
	//TODO: make sure that does not send done until all requests are finished
	// use another goroutine and check every request after the ticker is stopped
	s.TimeoutTimer.Stop()
	if (s.ReqLimitMode == "rate") {
		s.Ticker.Stop()
	}
	s.OverallTicker.Stop()
}

func (s *Spawner) SetupOverallStatsPipe() {
	Log("spawn", fmt.Sprintln("Overall stats will be gathered every ",time.Millisecond * overallStatsTickerFrequency) )

	s.OverallTicker = time.NewTicker(time.Millisecond * overallStatsTickerFrequency)
	go func () {
		for _ = range s.OverallTicker.C {
			s.SendOverallStats()
		}
	}()
}

func (s *Spawner) SendOverallStats() {
	overallStats := OverallStats {
		Rate : s.Rate,
		NumExecutors : len(s.ExecutorPool),
		StartTime : s.StartTime,
		RequestsIssued : s.RequestsIssued,
	}

	for _, executor := range s.ExecutorPool {
		if !executor.IsExecuting {
			overallStats.NumAvailableExecutors += 1
		} else {
			overallStats.NumBusyExecutors += 1
		}
	}

	overallStats.TimeElapsed = time.Since(s.StartTime)
	overallStats.TotalTestDuration = s.MaxExecutionTime

	s.OverallStatsChan <- overallStats
}

func (s *Spawner) SetupExecutorPool() {
	Log("spawn", fmt.Sprintln("Adding ", s.Concurrency ,"executors to pool") )

	s.ExecutorPool = make([]*Executor, s.Concurrency)

	for i:= 0; i < s.Concurrency; i++ {
		newExecutor := NewExecutor(fmt.Sprint(i), s.RequestChan, s.StatsChan, s.RequestOptions)

		if s.HasCustomClient() {
			newExecutor.CustomClient = s.CustomClient
		}

		go newExecutor.Start()

		s.ExecutorPool[i] = newExecutor
	}
}

func (s *Spawner) StartRequests() {
	if (s.ReqLimitMode == "total") {
		Log("spawn", fmt.Sprintln("Requests are limited by total quantity, ", s.RequestsToIssue, " requests have been buffered on the channel") )
		go func(){
			for i:= 0; i < s.RequestsToIssue; i++ {
				s.RequestsIssued += 1
				s.RequestChan <- true
			}
		}()
	} else if (s.ReqLimitMode == "rate") {
		s.Ticker = time.NewTicker(time.Second * tickerSecFrequency)
		go func(){
			for _ = range s.Ticker.C{
				Log("spawn", fmt.Sprintln(" Requests are rate limited - triggering set of ", s.Rate, " requests at ",time.Now() ) )
				s.mu.Lock()
				for i:= 0; i < int(s.Rate); i++ {
					if (s.RequestsIssued < s.RequestsToIssue) {
						s.RequestsIssued += 1
						s.RequestChan <- true
					} else {
						s.Cleanup()
						s.Stop()
						s.Done <- true
						break
					}
				}
				s.mu.Unlock()
			}

		}()
	}
}

func (s *Spawner) HasCustomClient() bool {
	if (s.CustomClient != nil && s.CustomClient.Transport != nil) {
		return true
	}
	return false
}
