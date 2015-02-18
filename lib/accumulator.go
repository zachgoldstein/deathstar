package lib

import (
	"sync"
)

type Accumulator struct {
	mu *sync.Mutex
	Stats []ResponseStats
	OverallStats []OverallStats

	StatsChan chan ResponseStats
	OverallStatsChan chan OverallStats
}

func NewAccumulator(statsChan chan ResponseStats, overallStatsChan chan OverallStats) *Accumulator {
	newAccumulator := &Accumulator{
		mu : &sync.Mutex{},
		StatsChan : statsChan,
		OverallStatsChan : overallStatsChan,
	}
	newAccumulator.Start()
	return newAccumulator
}

//Start will create a go routine to listen on channel for new stats
func (a *Accumulator)Start(){
	go func() {
		for stats := range a.StatsChan {
			a.mu.Lock()
			a.Stats = append(a.Stats, stats)
			a.mu.Unlock()
		}
	}()

	go func() {
		for stats := range a.OverallStatsChan {
			a.mu.Lock()
			a.OverallStats = append(a.OverallStats, stats)
			a.mu.Unlock()
		}
	}()

}
