package lib

import (
	"sync"
	"fmt"
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
//Setup will create a go routine to listen on channel for new stats
func (a *Accumulator)Start(){
	go func() {
		for stats := range a.StatsChan {
			a.mu.Lock()
			fmt.Println("Stats received, appending to slice", stats)
			fmt.Println("numExecutors", stats.NumExecutors)
			a.Stats = append(a.Stats, stats)
			a.mu.Unlock()
		}
	}()
}
