package lib

import (
	"sync"
)

type Reporter struct {
	DataChan chan AggregatedStats
	Done chan bool
	Pretty bool

	mu *sync.Mutex
	LatestData AggregatedStats

	LatestSummary string

	Renderer Renderer
}

type Renderer interface {
	//Implementing structs must store and send on this channel to indicate successful cleanup upon quit
	Setup(chan bool)
	Generate(stats AggregatedStats)
	Render()
	Quit()
}

func NewReporter(dataChan chan AggregatedStats, pretty bool) *Reporter {
	reporter := &Reporter{
		mu : &sync.Mutex{},
		DataChan : dataChan,
		Done : make(chan bool),
		Pretty : pretty,
	}

	if (reporter.Pretty) { //TODO: add cli option here
		reporter.Renderer = NewCliRenderer()
		reporter.Renderer.Setup(reporter.Done)
	}

	reporter.Start()

	return reporter
}

func (r *Reporter) Start() {
	go r.chanSetup()
}

func (r *Reporter) chanSetup() {
	counter := 0
	for data := range r.DataChan {
		counter += 1
		r.mu.Lock()
		r.LatestData = data
		r.Renderer.Generate(r.LatestData)

		if (r.Pretty) {
			r.Renderer.Render()
		}
		r.mu.Unlock()
	}
}

func (r *Reporter) Stop() {
	if (r.Pretty) {
		r.Renderer.Quit()
	}
}
