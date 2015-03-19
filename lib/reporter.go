package lib

import (
	"sync"
)

type Reporter struct {
	DataChan chan AggregatedStats
	Done chan bool
	RenderHTML bool
	RenderCLI bool

	mu *sync.Mutex
	LatestData AggregatedStats

	LatestSummary string

	Renderers []Renderer
}

type Renderer interface {
	//Implementing structs must store and send on this channel to indicate successful cleanup upon quit
	Setup(chan bool)
	Generate(stats AggregatedStats)
	Render()
	Quit()
}

func NewReporter(dataChan chan AggregatedStats, opts OutputOptions, reqOpts RequestOptions) *Reporter {
	reporter := &Reporter{
		mu : &sync.Mutex{},
		DataChan : dataChan,
		Done : make(chan bool),
		RenderHTML : opts.ShowHTML,
		RenderCLI : opts.ShowCLI,
	}

	//TODO: add support for multiple renderers at one time
	if reporter.RenderHTML {
		renderer := NewRenderHTML(reqOpts)
		renderer.Setup(reporter.Done)
		reporter.Renderers = append(reporter.Renderers, renderer)
		reporter.Start()
	}

	if reporter.RenderCLI {
		renderer := NewRenderCLI(reqOpts)
		renderer.Setup(reporter.Done)
		reporter.Renderers = append(reporter.Renderers, renderer)
		reporter.Start()
	}

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
		for _, renderer := range r.Renderers {
			renderer.Generate(r.LatestData)
			renderer.Render()
		}

		r.mu.Unlock()
	}
}

func (r *Reporter) Cleanup() {
	for _, renderer := range r.Renderers {
		renderer.Generate(r.LatestData)
		renderer.Render()
	}
}

func (r *Reporter) Stop() {
	if (r.RenderCLI) {
		for _, renderer := range r.Renderers {
			renderer.Quit()
		}
	}
}
