package lib

import (
	"log"
	"github.com/jroimartin/gocui"
	"fmt"
	"github.com/aybabtme/uniplot/histogram"
	"bytes"
	"time"
	"io/ioutil"
	"errors"
	"sort"
	"github.com/cheggaaa/pb"
)

type CLIRenderData struct {
	Latest AggregatedStats

	LatestConnectPercentiles string
	LatestTotalPercentiles string
	LatestResponsePercentiles string

	LatestConnectHistogram string
	LatestTotalHistogram string
	LatestResponseHistogram string
	LatestProgress string

	LatestFailures []string

	LatestSummary string

	LatestTopPercentile string
}

var title = `
    ____             __  __         __
   / __ \___  ____ _/ /_/ /_  _____/ /_____ ______
  / / / / _ \/ __ '/ __/ __ \/ ___/ __/ __ '/ ___/
 / /_/ /  __/ /_/ / /_/ / / (__  ) /_/ /_/ / /
/_____/\___/\__,_/\__/_/ /_/____/\__/\__,_/_/
                                                  `

func NewRenderCLI(reqOpts RequestOptions) *RenderCLI {
	return &RenderCLI{
		ReqOpts : reqOpts,
		Done : make(chan bool),
	}
}

type RenderCLI struct {
	GUI *gocui.Gui
	ReqOpts RequestOptions
	Data CLIRenderData
	Done chan bool

	IsClosed bool
}

func (r *RenderCLI)Setup(done chan bool)  {
	r.Done = done
	go func() {
		var err error
		r.GUI = gocui.NewGui()
		if err := r.GUI.Init(); err != nil {
			log.Panicln(err)
		}
		defer r.GUI.Close()
		r.GUI.SetLayout(r.renderGUI)
		if err := r.GUI.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, r.quitGUI); err != nil {
			log.Panicln(err)
		}
		err = r.GUI.MainLoop()
		if err != nil && err != gocui.Quit {
			log.Panicln(err)
		}
	} ()
}

func (r *RenderCLI)Generate(stats AggregatedStats) {
	if (r.IsClosed) { return }
	r.Data.Latest = stats

	r.Data.LatestConnectPercentiles, r.Data.LatestTotalPercentiles, r.Data.LatestResponsePercentiles = r.GeneratePercentiles(stats)
	r.Data.LatestConnectHistogram, r.Data.LatestTotalHistogram, r.Data.LatestResponseHistogram = r.GenerateHistogram(stats)
	r.Data.LatestProgress = r.GenerateProgressBar(stats)
	r.Data.LatestFailures = r.GenerateFailures(stats)

	if (len(stats.TotalTimePercentiles) > 0) {
		r.Data.LatestTopPercentile = stats.TotalTimePercentiles[ len(stats.TotalTimePercentiles) -1 ].String()
	}
}

func (r *RenderCLI)Render() {
	if (!r.IsClosed) {
		r.GUI.SetLayout(r.renderGUI)
	}
}

func (r *RenderCLI)Quit() {
	if (r.IsClosed) {
		return
	}
	err := r.renderGUI(r.GUI)
	if (err != nil) {
		log.Panicln(err)
	}
	r.GUI.Close()
	r.IsClosed = true
}

func (r *RenderCLI)renderGUI(g *gocui.Gui) error{
	maxX, maxY := g.Size()

	leftView, err := g.SetView("left", 0, 24, maxX/3, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(leftView, "Response Times")
	fmt.Fprintln(leftView, r.Data.LatestResponsePercentiles)
	fmt.Fprintln(leftView, r.Data.LatestResponseHistogram)

	rightView, err := g.SetView("right", maxX*2/3, 24, maxX-1, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(rightView, "Total Response Times")
	fmt.Fprintln(rightView, r.Data.LatestTotalPercentiles)
	fmt.Fprintln(rightView, r.Data.LatestTotalHistogram)

	middleView, err := g.SetView("middle", maxX/3, 24, maxX*2/3, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(middleView, "Connect Times")
	fmt.Fprintln(middleView, r.Data.LatestConnectPercentiles)
	fmt.Fprintln(middleView, r.Data.LatestConnectHistogram)

	topLeftView, err := g.SetView("topLeftView", 0, 9, maxX/2, 23)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(topLeftView, "Summary")
	fmt.Fprintln(topLeftView, "Requests to Issue: ", r.ReqOpts.RequestsToIssue)
	fmt.Fprintln(topLeftView, "Requests Issued: ", r.Data.Latest.TotalRequests)
	fmt.Fprintln(topLeftView, "Failures: ", r.Data.Latest.Failures)
	fmt.Fprintln(topLeftView, "Maximum Response Time: ", r.Data.Latest.MaxTotalTime)
	if (len(r.Data.LatestTotalPercentiles) > 0 && len(r.Data.Latest.Percentiles) > 0) {
		fmt.Fprintln(topLeftView, r.Data.Latest.Percentiles[len(r.Data.Latest.Percentiles) - 1] * 100, "th Percentile time: ",  r.Data.LatestTopPercentile)
	}
	fmt.Fprintln(topLeftView, "Minimum Response Time: ", r.Data.Latest.MinTotalTime)
	fmt.Fprintln(topLeftView, "Started at, ", r.Data.Latest.StartTime)
	fmt.Fprintln(topLeftView, "Run for, ", r.Data.Latest.TimeElapsed)
	fmt.Fprintln(topLeftView, "Total Running Time ", r.Data.Latest.TotalTestDuration)

	topRightView, err := g.SetView("topRightView", maxX/2, 9, maxX-1, 23)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	for _, failure := range r.Data.LatestFailures {
		fmt.Fprintln(topRightView, failure)
	}
	topRightView.Wrap = true

	topProgress, err := g.SetView("topProgress", 0, 7, maxX-1, 9)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(topProgress, r.Data.LatestProgress)

	titleView, err := g.SetView("titleView", maxX/2-26, 0, maxX/2+26, 6)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(titleView, title)

	return nil
}

func (r *RenderCLI)quitGUI(g *gocui.Gui, v *gocui.View) error{
	r.Quit()

	r.Done <- true
	return gocui.Quit
}

func (r *RenderCLI) GenerateFailures(stats AggregatedStats) (failuresStrs []string) {
	for _, failures := range r.Data.Latest.FailureCounts {
		if (len(failures) > 1) {
			failuresStrs = append(failuresStrs, fmt.Sprintf("%v Failures: %v", len(failures), failures[0].Error()) )
		}
	}
	sort.Strings(failuresStrs)
	return failuresStrs
}

func (r *RenderCLI) GenerateProgressBar(stats AggregatedStats) string {
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

func (r *RenderCLI) GenerateHistogram(stats AggregatedStats) (connectOutput, totalOutput, responseOutput string){
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

func (r *RenderCLI) GeneratePercentiles(stats AggregatedStats) (connectOutput, totalOutput, responseOutput string){
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
