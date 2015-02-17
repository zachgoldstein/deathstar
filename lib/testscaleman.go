package lib

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

func DoScaleTest() {

	/*
	Pseudocode for flow
	Digest command line params & config json file to generate http.Request
	Setup a spawner, which will initiate requests on a channel at a specific rate
	Setup a pool of executors, according to the concurrency, which issue the requests
	Setup an accumulator, which receives all responses and stores their stats
	Create a channel for the spawner, executor pool and accumulator to use.

	Setup an analyser, which periodically scans the accumulator and generates meaningful aggregated stats
	Setup a reporter, which renders the aggregated stats to stdOut and a live-updating page.
	 */

	reqOpts, outOpts, err := digestOptions()
	Log("debug", fmt.Sprintf("reqOpts,",reqOpts, "outOpts, ", outOpts) )
	if (err != nil) {
		issueError(err)
	}

	responseStatsChan := make(chan ResponseStats)

	maxTestTime := time.Second * 5

	spawner := NewSpawner(3, maxTestTime, responseStatsChan, reqOpts)
	accumulator := NewAccumulator(spawner.StatsChan)
	spawner.Start()

	reportFrequency := time.Second * 1
	percentiles := []float64{0.01, 0.05, 0.25, 0.50, 0.75, 0.95, 0.99, 0.999, 0.9999}
	analyser := NewAnalyser(accumulator, reportFrequency, percentiles)
	NewReporter(analyser.StatsChan)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	Log("debug", fmt.Sprintf("Main blocking flow") )

	for {
		select {
		case <-spawner.Done:
			Log("debug", fmt.Sprintf("Completed execution") )
			os.Exit(0)
		case <-c:
			Log("debug", fmt.Sprintf("Interupted, exiting") )
			os.Exit(1)
		}
	}
}

//issueError will print an error to stdOut that is better formatted than a normal panic
func issueError(err error) {
	panic(err)
}
