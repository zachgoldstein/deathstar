package lib

import (
	"fmt"
	"os"
	"os/signal"
	"time"
	"runtime"
)

func DoScaleTest() {

	reqOpts, outOpts, err := digestOptions()
	Log("top", fmt.Sprintf("reqOpts,",reqOpts, "outOpts, ", outOpts) )
	if (err != nil) {
		issueError(err)
	}

	runtime.GOMAXPROCS(4)

	responseStatsChan := make(chan ResponseStats)
	overallStatsChan := make(chan OverallStats)

	maxTestTime := time.Second * 180

	spawner := NewSpawner(100, maxTestTime, responseStatsChan, overallStatsChan, reqOpts)
	accumulator := NewAccumulator(spawner.StatsChan, spawner.OverallStatsChan)
	spawner.Start()

	reportFrequency := time.Millisecond * 100
	percentiles := []float64{0.01, 0.05, 0.25, 0.50, 0.75, 0.95, 0.99, 0.999, 0.9999}
	analyser := NewAnalyser(accumulator, reportFrequency, percentiles)
	reporter := NewReporter(analyser.StatsChan, false)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	Log("top", fmt.Sprintf("Main blocking flow") )

	for {
		select {
		case <-spawner.Done:
			reporter.Stop()
			Log("top", fmt.Sprintf("Completed execution") )
			os.Exit(0)
		case <-reporter.Done:
			Log("top", fmt.Sprintf("Interupted, exiting") )
			os.Exit(1)
		case <-c:
			Log("top", fmt.Sprintf("Interupted, exiting") )
			os.Exit(1)
		}
	}
}

//issueError will print an error to stdOut that is better formatted than a normal panic
func issueError(err error) {
	panic(err)
}
