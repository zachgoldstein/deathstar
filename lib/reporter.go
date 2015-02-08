package lib

import "fmt"

type Reporter struct {
	DataChan chan AggregatedStats
}

func NewReporter(dataChan chan AggregatedStats) *Reporter {
	reporter := &Reporter{
		DataChan : dataChan,
	}

	reporter.Start()

	return reporter
}

func (r *Reporter) Start() {
	go r.chanSetup()
}

func (r *Reporter) chanSetup() {
	for data := range r.DataChan {
		fmt.Print( r.GenerateReport(data) )
	}
}

func (r *Reporter) GenerateReport(stats AggregatedStats) string {
	return fmt.Sprintln("REPORT ON STATS: ",stats)
}
