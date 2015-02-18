package lib

import (
	"testing"
	"time"
	"fmt"
)

func TestNewAnalyser(t *testing.T) {
	responseStatsChan := make(chan ResponseStats)
	overallStatsChan := make(chan OverallStats)
	accumulator := NewAccumulator(responseStatsChan, overallStatsChan)

	analyser := NewAnalyser(accumulator, time.Millisecond * 500, []float64{0.01, 0.05, 0.25, 0.50, 0.75, 0.95, 0.99, 0.999, 0.9999} )
	go func(){
		for data := range analyser.StatsChan {
			fmt.Printf("pulled off statsChan ",data)
		}
	}()
	fmt.Printf("CREATED ANALYSER")
	go analyser.Start()
	fmt.Printf("STARTED ANALYSER")

	time.Sleep(time.Second * 5)
	fmt.Printf("FINISHED SLEEP")
}


