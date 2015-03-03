package lib

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

func DoScaleTest() {

	reqOpts, outOpts, err := digestOptions()
	if (err != nil) {
		issueError(err)
	}

	responseStatsChan := make(chan ResponseStats)
	overallStatsChan := make(chan OverallStats)
	Log("top", fmt.Sprintf("reqOpts.RequestsToIssue ",reqOpts.RequestsToIssue) )

	spawner := NewSpawner(responseStatsChan, overallStatsChan, reqOpts)
	accumulator := NewAccumulator(reqOpts.RequestsToIssue, spawner.StatsChan, spawner.OverallStatsChan)

	reportFrequency := time.Millisecond * 200
	percentiles := []float64{0.01, 0.05, 0.25, 0.50, 0.75, 0.95, 0.99, 0.999, 0.9999}
	analyser := NewAnalyser(accumulator, reportFrequency, percentiles)
	reporter := NewReporter(analyser.StatsChan, false, outOpts, reqOpts)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	Log("top", fmt.Sprintf("Executing!") )
	spawner.Start()

	Log("top", fmt.Sprintf("Main Loop!") )
	for {
		select {
		case <-spawner.Done:
			cleanUp(spawner, analyser, reporter)
			Log("top", fmt.Sprintf("Max execution time reached") )
			os.Exit(0)
		case <-accumulator.Done:
			cleanUp(spawner, analyser, reporter)
			Log("top", fmt.Sprintf("Finished executing all requests, exiting") )
			os.Exit(0)
		case <-reporter.Done:
			cleanUp(spawner, analyser, reporter)
			Log("top", fmt.Sprintf("Interupted, exiting") )
			os.Exit(1)
		case <-c:
			cleanUp(spawner, analyser, reporter)
			Log("top", fmt.Sprintf("Interupted, exiting") )
			os.Exit(1)
		}
	}
}

func cleanUp(spawner *Spawner, analyser *Analyser, reporter *Reporter) {
	spawner.Stop()
	analyser.Stop()

	spawner.Cleanup()
	analyser.Cleanup()

	reporter.Cleanup()
	reporter.Stop()
}

//issueError will print an error to stdOut that is better formatted than a normal panic
func issueError(err error) {
	panic(err)
}
