package lib

import (
	"fmt"
	"github.com/aybabtme/uniplot/histogram"
	"time"
	"io/ioutil"
	"bytes"
	"github.com/jroimartin/gocui"
	"sync"
	"log"
	"errors"
	"github.com/cheggaaa/pb"
	"sort"
)

type Reporter struct {
	DataChan chan AggregatedStats
	Done chan bool
	GUI *gocui.Gui
	Pretty bool

	mu *sync.Mutex
	LatestHistogram string

	LatestConnectPercentiles string
	LatestTotalPercentiles string
	LatestResponsePercentiles string

	LatestConnectHistogram string
	LatestTotalHistogram string
	LatestResponseHistogram string
	LatestProgress string
	LatestFailures []string

	LatestData AggregatedStats

	LatestSummary string
}

func NewReporter(dataChan chan AggregatedStats, pretty bool) *Reporter {
	reporter := &Reporter{
		mu : &sync.Mutex{},
		DataChan : dataChan,
		Done : make(chan bool),
		Pretty : pretty,
	}

	reporter.Start()
	if (reporter.Pretty) {
		go reporter.SetupRenderer()
	}

	return reporter
}

func (r *Reporter) SetupRenderer() {
	var err error
	r.GUI = gocui.NewGui()
	if err := r.GUI.Init(); err != nil {
		log.Panicln(err)
	}
	defer r.GUI.Close()
	r.GUI.SetLayout(r.Render)
	if err := r.GUI.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, r.quit); err != nil {
		log.Panicln(err)
	}
	err = r.GUI.MainLoop()
	if err != nil && err != gocui.Quit {
		log.Panicln(err)
	}
}

func (r *Reporter) Start() {
	go r.chanSetup()
}

func (r *Reporter) chanSetup() {
	counter := 0
	for data := range r.DataChan {
		counter += 1
//		for i:= 0; i < 50; i++ {
//			fmt.Fprintf(os.Stdout, "   \r")
//		}
//		fmt.Fprintf(os.Stdout, "%v \n", counter)
		r.mu.Lock()
		r.GenerateReport(data)
		r.LatestData = data
		if (r.Pretty) {
			r.GUI.SetLayout(r.Render)
		}
		r.mu.Unlock()
//		fmt.Fprintf(os.Stdout, "1. this is a test of new lines")
//		fmt.Fprintf(os.Stdout, "2. this is a test of new lines")
		//		r.RefreshingLog(r.GenerateReport(data))
	}
}

func (r *Reporter) GenerateReport(stats AggregatedStats)  {
	r.LatestConnectPercentiles, r.LatestTotalPercentiles, r.LatestResponsePercentiles = r.GeneratePercentiles(stats)
	r.LatestConnectHistogram, r.LatestTotalHistogram, r.LatestResponseHistogram = r.GenerateHistogram(stats)
	r.LatestProgress = r.GenerateProgressBar(stats)
	r.LatestFailures = r.GenerateFailures(stats)
}

func (r *Reporter) GenerateFailures(stats AggregatedStats) (failures []string) {
	for title, count := range stats.FailureCounts {
		failures = append(failures, fmt.Sprintf("%v Failures: %v", count, title) )
	}
	sort.Strings(failures)
	return failures
}

func (r *Reporter) GenerateProgressBar(stats AggregatedStats) string {
	count := int(stats.TotalTestDuration.Nanoseconds())
	bar := pb.StartNew(count)
	bar.NotPrint = true
	output := bytes.NewBuffer([]byte{})
	bar.Output = output
	bar.ShowCounters = false

	progress := stats.TimeElapsed.Nanoseconds()
	bar.Set(int(progress))
	bar.Finish()

	bytes, _ := ioutil.ReadAll(output)

	return string(bytes)
}

func (r *Reporter) GenerateHistogram(stats AggregatedStats) (connectOutput, totalOutput, responseOutput string){
	if (stats.TotalRequests == 0){
		output := "No requests returned yet...."
		return output, output, output
	}

	connectOutput = "Histogram: \n"
	totalOutput = "Histogram: \n"
	responseOutput = "Histogram: \n"

	output, _ := getHist(stats.TimeToConnect)
	connectOutput += output
	output, _ = getHist(stats.TotalTime)
	totalOutput += output
	output, _ = getHist(stats.TimeToRespond)
	responseOutput += output

	return
}

func getHist(data []float64) (output string, err error) {
	if (len(data) == 0) {
		return "", errors.New("No data found to create histogram")
	}
	hist := histogram.PowerHist(2.0, data)

	strBuf := bytes.NewBuffer([]byte{})

	err = histogram.Fprintf(strBuf, hist, histogram.Linear(5), func(v float64) string {
		return time.Duration(v).String()
	})
	if err != nil {
		return "", err
	}

	bytes, err := ioutil.ReadAll(strBuf)

	return string(bytes), nil
}

func (r *Reporter) GeneratePercentiles(stats AggregatedStats) (connectOutput, totalOutput, responseOutput string){
	if (stats.TotalRequests == 0){
		output := "No requests returned yet...."
		return output, output, output
	}

	connectOutput = "Percentiles: \n"
	totalOutput = "Percentiles: \n"
	responseOutput = "Percentiles: \n"

	for index, percentile := range stats.Percentiles {
		if (index == 0) {
			connectOutput += fmt.Sprintf("%vst Percentile: %s \n",percentile*100, stats.TimeToConnectPercentiles[index].String())
			totalOutput += fmt.Sprintf("%vst Percentile: %s \n",percentile*100, stats.TotalTimePercentiles[index].String())
			responseOutput += fmt.Sprintf("%vst Percentile: %s \n",percentile*100, stats.TimeToRespondPercentiles[index].String())
		} else {
			connectOutput += fmt.Sprintf("%vth Percentile: %s \n",percentile*100, stats.TimeToConnectPercentiles[index].String())
			totalOutput += fmt.Sprintf("%vth Percentile: %s \n",percentile*100, stats.TotalTimePercentiles[index].String())
			responseOutput += fmt.Sprintf("%vth Percentile: %s \n",percentile*100, stats.TimeToRespondPercentiles[index].String())
		}
	}

	return connectOutput, totalOutput, responseOutput
}


/*
Good output example

% boom -n 1000 -c 100 https://google.com
1000 / 1000 ∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎ 100.00 %

Summary:
  Total:        21.1307 secs.
  Slowest:      2.9959 secs.
  Fastest:      0.9868 secs.
  Average:      2.0827 secs.
  Requests/sec: 47.3246
  Speed index:  Hahahaha

Response time histogram:
  0.987 [1]     |
  1.188 [2]     |
  1.389 [3]     |
  1.590 [18]    |∎∎
  1.790 [85]    |∎∎∎∎∎∎∎∎∎∎∎
  1.991 [244]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  2.192 [284]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  2.393 [304]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  2.594 [50]    |∎∎∎∎∎∎
  2.795 [5]     |
  2.996 [4]     |

Latency distribution:
  10% in 1.7607 secs.
  25% in 1.9770 secs.
  50% in 2.0961 secs.
  75% in 2.2385 secs.
  90% in 2.3681 secs.
  95% in 2.4451 secs.
  99% in 2.5393 secs.

Status code distribution:
  [200] 1000 responses

 */


func (r *Reporter) Render(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	leftView, err := g.SetView("left", 0, 15, maxX/3, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(leftView, "Response Times")
	fmt.Fprintln(leftView, r.LatestResponsePercentiles)
	fmt.Fprintln(leftView, r.LatestResponseHistogram)

	rightView, err := g.SetView("right", maxX*2/3, 15, maxX-1, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(rightView, "Total Response Times")
	fmt.Fprintln(rightView, r.LatestTotalPercentiles)
	fmt.Fprintln(rightView, r.LatestTotalHistogram)

	middleView, err := g.SetView("middle", maxX/3, 15, maxX*2/3, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(middleView, "Connect Times")
	fmt.Fprintln(middleView, r.LatestConnectPercentiles)
	fmt.Fprintln(middleView, r.LatestConnectHistogram)

	topLeftView, err := g.SetView("topLeftView", 0, 6, maxX/2, 15)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(topLeftView, "Summary")
	fmt.Fprintln(topLeftView, "Total Request: ", r.LatestData.TotalRequests)
	fmt.Fprintln(topLeftView, "Failures: ", r.LatestData.Failures)
	fmt.Fprintln(topLeftView, "Maximum Response Time: ", r.LatestData.MaxTotalTime)
	fmt.Fprintln(topLeftView, "Minimum Response Time: ", r.LatestData.MinTotalTime)
	fmt.Fprintln(topLeftView, "Started at, ", r.LatestData.StartTime)
	fmt.Fprintln(topLeftView, "Run for, ", r.LatestData.TimeElapsed)
	fmt.Fprintln(topLeftView, "Total Running Time ", r.LatestData.TotalTestDuration)

	topRightView, err := g.SetView("topRightView", maxX/2, 6, maxX-1, 15)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
//	fmt.Fprintln(topRightView, "Failures", r.LatestFailures)
	for _, failure := range r.LatestFailures {
		fmt.Fprintln(topRightView, failure)
	}
	topRightView.Wrap = true

	topProgress, err := g.SetView("topProgress", 0, 3, maxX-1, 5)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(topProgress, r.LatestProgress)

	titleView, err := g.SetView("titleView", maxX/2-8, 0, maxX/2+8, 2)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(titleView, "Testman Results")

	return nil
}

func (r *Reporter) Stop() {
	if (r.Pretty) {
		r.Render(r.GUI)
		r.GUI.Close()
	}

//	fmt.Printf("FINAL STATS: %v", r.LatestData)
}

func (r *Reporter) quit(g *gocui.Gui, v *gocui.View) error {
	r.Stop()

	r.Done <- true
	return gocui.Quit
}
