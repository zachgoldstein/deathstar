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
}

func NewCliRenderer() *RenderCLI {
	return &RenderCLI{
		Done : make(chan bool),
	}
}

type RenderCLI struct {
	Renderer
	GUI *gocui.Gui
	Data CLIRenderData
	Done chan bool
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
	r.Data.Latest = stats

	r.Data.LatestConnectPercentiles, r.Data.LatestTotalPercentiles, r.Data.LatestResponsePercentiles = r.GeneratePercentiles(stats)
	r.Data.LatestConnectHistogram, r.Data.LatestTotalHistogram, r.Data.LatestResponseHistogram = r.GenerateHistogram(stats)
	r.Data.LatestProgress = r.GenerateProgressBar(stats)
	r.Data.LatestFailures = r.GenerateFailures(stats)
}

func (r *RenderCLI)Render() {
	r.GUI.SetLayout(r.renderGUI)
}

func (r *RenderCLI)Quit() {
	r.renderGUI(r.GUI)
	r.GUI.Close()
}

func (r *RenderCLI)renderGUI(g *gocui.Gui) error{
	maxX, maxY := g.Size()

	leftView, err := g.SetView("left", 0, 15, maxX/3, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(leftView, "Response Times")
	fmt.Fprintln(leftView, r.Data.LatestResponsePercentiles)
	fmt.Fprintln(leftView, r.Data.LatestResponseHistogram)

	rightView, err := g.SetView("right", maxX*2/3, 15, maxX-1, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(rightView, "Total Response Times")
	fmt.Fprintln(rightView, r.Data.LatestTotalPercentiles)
	fmt.Fprintln(rightView, r.Data.LatestTotalHistogram)

	middleView, err := g.SetView("middle", maxX/3, 15, maxX*2/3, maxY-1)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(middleView, "Connect Times")
	fmt.Fprintln(middleView, r.Data.LatestConnectPercentiles)
	fmt.Fprintln(middleView, r.Data.LatestConnectHistogram)

	topLeftView, err := g.SetView("topLeftView", 0, 6, maxX/2, 15)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(topLeftView, "Summary")
	fmt.Fprintln(topLeftView, "Total Request: ", r.Data.Latest.TotalRequests)
	fmt.Fprintln(topLeftView, "Failures: ", r.Data.Latest.Failures)
	fmt.Fprintln(topLeftView, "Maximum Response Time: ", r.Data.Latest.MaxTotalTime)
	fmt.Fprintln(topLeftView, "Minimum Response Time: ", r.Data.Latest.MinTotalTime)
	fmt.Fprintln(topLeftView, "Started at, ", r.Data.Latest.StartTime)
	fmt.Fprintln(topLeftView, "Run for, ", r.Data.Latest.TimeElapsed)
	fmt.Fprintln(topLeftView, "Total Running Time ", r.Data.Latest.TotalTestDuration)

	topRightView, err := g.SetView("topRightView", maxX/2, 6, maxX-1, 15)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	for _, failure := range r.Data.LatestFailures {
		fmt.Fprintln(topRightView, failure)
	}
	topRightView.Wrap = true

	topProgress, err := g.SetView("topProgress", 0, 3, maxX-1, 5)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(topProgress, r.Data.LatestProgress)

	titleView, err := g.SetView("titleView", maxX/2-8, 0, maxX/2+8, 2)
	if err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
	}
	fmt.Fprintln(titleView, "Testman Results")

	return nil
}

func (r *RenderCLI)quitGUI(g *gocui.Gui, v *gocui.View) error{
	r.Quit()

	r.Done <- true
	return gocui.Quit
}

func (r *RenderCLI) GenerateFailures(stats AggregatedStats) (failures []string) {
	for title, count := range stats.FailureCounts {
		failures = append(failures, fmt.Sprintf("%v Failures: %v", count, title) )
	}
	sort.Strings(failures)
	return failures
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
