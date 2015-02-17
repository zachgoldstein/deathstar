package lib

import (
	"sync"
)

type Accumulator struct {
	mu *sync.Mutex
	Stats []ResponseStats

	StatsChan chan ResponseStats
}

func NewAccumulator(statsChan chan ResponseStats) *Accumulator {
	newAccumulator := &Accumulator{
		mu : &sync.Mutex{},
		StatsChan : statsChan,
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
}
