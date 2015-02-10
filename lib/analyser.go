package lib

import (
	"time"
	"fmt"
)

//Analyser will
type Analyser struct {
	Frequency time.Duration
	Ticker *time.Ticker
	Accumulator *Accumulator
	StatsChan chan AggregatedStats
}

type AggregatedStats struct {
	TotalRequests int
	ConcurrentExecutors int
}

func NewAnalyser(acc *Accumulator, frequency time.Duration) (*Analyser) {
	analyser := &Analyser{
		Accumulator : acc,
		Frequency : frequency,
		StatsChan : make(chan AggregatedStats),
	}
	analyser.Start()
	return analyser
}

func (a *Analyser) Start() {
	a.Ticker = time.NewTicker(a.Frequency)
	go a.Analyse()
}

func (a *Analyser) Analyse() {
	for _ = range a.Ticker.C {
		fmt.Println("Analysing mock")
		stats := AggregatedStats{
			TotalRequests : len(a.Accumulator.Stats),
		}
		fmt.Println("Performed analysis and sent to channel ",stats)
		a.StatsChan <- stats
	}
}
