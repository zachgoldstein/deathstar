package lib

import (
	"fmt"
	"github.com/aybabtme/uniplot/histogram"
	"time"
	"io/ioutil"
	"bytes"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"github.com/jroimartin/gocui"
	"sync"
)

type Reporter struct {
	DataChan chan AggregatedStats
	OriginalState *terminal.State
	GUI *gocui.Gui

	mu *sync.Mutex
	LatestHistogram string

	LatestConnectPercentiles string
	LatestTotalPercentiles string
	LatestResponsePercentiles string

	LatestConnectHistogram string
	LatestTotalHistogram string
	LatestResponseHistogram string

	LatestSummary string
}

func NewReporter(dataChan chan AggregatedStats) *Reporter {
	reporter := &Reporter{
		mu : &sync.Mutex{},
		DataChan : dataChan,
	}

	reporter.Start()
	go reporter.SetupRenderer()

	return reporter
}

func (r *Reporter) SetupRenderer() {
	var err error
	r.GUI = gocui.NewGui()
	if err := r.GUI.Init(); err != nil {
		log.Panicln(err)
	}
	defer r.GUI.Close()
	r.GUI.SetLayout(r.layout)
	if err := r.GUI.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	err = r.GUI.MainLoop()
	if err != nil && err != gocui.Quit {
		log.Panicln(err)
	}
}

func (r *Reporter) Start() {
	r.OriginalState, _ = terminal.MakeRaw(0)

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
		r.GUI.SetLayout(r.layout)
		r.mu.Unlock()
//		fmt.Fprintf(os.Stdout, "1. this is a test of new lines")
//		fmt.Fprintf(os.Stdout, "2. this is a test of new lines")
		//		r.RefreshingLog(r.GenerateReport(data))
	}
}

func (r *Reporter) RefreshingLog(output string) {
//	oldState, err := terminal.MakeRaw(0)
//	if err != nil {
//		panic(err)
//	}
//	defer terminal.Restore(0, oldState)
//	Log("all", "\x0c  ", output)
	fmt.Println(output)
}

func (r *Reporter) GenerateReport(stats AggregatedStats) {
	r.LatestConnectPercentiles, r.LatestTotalPercentiles, r.LatestResponsePercentiles = r.GeneratePercentiles(stats)
	r.LatestConnectHistogram, r.LatestTotalHistogram, r.LatestResponseHistogram = r.GenerateHistogram(stats)
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
	hist := histogram.PowerHist(2, data)

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


func (r *Reporter) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	leftView, err := g.SetView("left", 0, 13, maxX/3, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(leftView, "Response Times")
	fmt.Fprintln(leftView, r.LatestResponsePercentiles)
	fmt.Fprintln(leftView, r.LatestResponseHistogram)

	rightView, err := g.SetView("right", maxX*2/3, 13, maxX-1, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(rightView, "Total Response Times")
	fmt.Fprintln(rightView, r.LatestTotalPercentiles)
	fmt.Fprintln(rightView, r.LatestTotalHistogram)

	middleView, err := g.SetView("middle", maxX/3, 13, maxX*2/3, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(middleView, "Connect Times")
	fmt.Fprintln(middleView, r.LatestConnectPercentiles)
	fmt.Fprintln(middleView, r.LatestConnectHistogram)

	topView, err := g.SetView("topView", 0, 3, maxX-1, 13)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(topView, "Summary")
	fmt.Fprintln(topView, "-")
	fmt.Fprintln(topView, "-")
	fmt.Fprintln(topView, "-")
	fmt.Fprintln(topView, "-")
	fmt.Fprintln(topView, "-")


	titleView, err := g.SetView("titleView", maxX/2-8, 0, maxX/2+8, 2)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(titleView, "Testman Results")

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.Quit
}
