package lib

import (
	"os"
	"os/signal"
	"fmt"
	"time"
)

type Choreographer struct {
	ExecuteSingleRequest bool
	IncreaseRateToFailure bool

	RequestOptions RequestOptions
	OutputOptions OutputOptions

	ResponseStatsChan chan ResponseStats
	OverallStatsChan chan OverallStats

	Spawner *Spawner
	Accumulator *Accumulator
	Analyser *Analyser
	Reporter *Reporter

}

func NewChoreographer(reqOpts RequestOptions, outOpts OutputOptions) *Choreographer{
	choreographer := &Choreographer{
		ExecuteSingleRequest : reqOpts.ExecuteSingleRequest,
		IncreaseRateToFailure : reqOpts.IncreaseRateToFailure,
		RequestOptions : reqOpts,
		OutputOptions : outOpts,
		ResponseStatsChan : make(chan ResponseStats),
		OverallStatsChan : make(chan OverallStats),
	}

	choreographer.Spawner = NewSpawner(choreographer.ResponseStatsChan, choreographer.OverallStatsChan, choreographer.RequestOptions)
	choreographer.Accumulator = NewAccumulator(choreographer.RequestOptions.RequestsToIssue, choreographer.Spawner.StatsChan, choreographer.Spawner.OverallStatsChan)

	calcRate := false
	if (!choreographer.IncreaseRateToFailure) {
		calcRate = true
	}

	choreographer.Analyser = NewAnalyser(choreographer.Accumulator, reqOpts, calcRate)
	choreographer.Reporter = NewReporter(choreographer.Analyser.StatsChan, choreographer.OutputOptions, choreographer.RequestOptions)

	if (choreographer.ExecuteSingleRequest) {
		choreographer.Spawner.RequestsToIssue = 1
	}

	return choreographer
}

func (c *Choreographer) Start() {
	Log("top", fmt.Sprintf("Starting to execute") )

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	c.Spawner.Start()

	now := time.Now()
	for {
		select {
		case <- c.Analyser.Fail:
			if (c.IncreaseRateToFailure) {
				c.cleanup()
				Log("top", fmt.Sprintf("Failure occurred at %v", time.Since(now)) )
				os.Exit(0)
			}
		case <- c.Spawner.Done:
			c.cleanup()
			Log("top", fmt.Sprintf("Max execution time reached") )
			os.Exit(0)
		case <- c.Accumulator.Done:
			c.cleanup()
			Log("top", fmt.Sprintf("Finished executing all requests, exiting") )
			os.Exit(0)
		case <- c.Reporter.Done:
			c.cleanup()
			Log("top", fmt.Sprintf("Interupted, exiting") )
			os.Exit(1)
		case <- sigChan:
			c.cleanup()
			Log("top", fmt.Sprintf("Interupted, exiting") )
			os.Exit(1)
		}
	}
}

func (c *Choreographer) cleanup () {
	c.Spawner.Stop()
	c.Analyser.Stop()

	c.Spawner.Cleanup()
	c.Analyser.Cleanup()

	c.Reporter.Cleanup()
	c.Reporter.Stop()
}
