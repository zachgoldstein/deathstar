package lib

import (
	"net/http"
	"log"
	"html/template"
	"io/ioutil"
	"bytes"
	"fmt"
)

type RenderHTML struct {
	Port int
	Done chan bool
	Data RenderData
}

type RenderData struct {
	Latest AggregatedStats
	Title string

	LatestConnectPercentiles []float64
 	LatestTotalPercentiles []float64
 	LatestResponsePercentiles []float64
	PercentileTitles []string

	TotalHistogramTimes []float64
	ConnectionHistogramTimes []float64

	ProgressBarMax int64
	ProgressBarCurrent int64
	PercentageComplete string
}

func NewRenderHTML() *RenderHTML {
	return &RenderHTML{
		Data : RenderData{},
	}
}

func (r *RenderHTML) Setup(done chan bool) {
	r.Done = done
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write( []byte("THIS IS A TEST") )
			if (err != nil){
				http.Error(w, "Could not render view", 500)
			}
		})
		log.Print("SERVING ON PORT 9090")
		log.Fatal(http.ListenAndServe(":9090", nil))
	}()
}

func (r *RenderHTML) Generate(stats AggregatedStats) {
	r.Data.Title = "this is a test"
	r.Data.Latest = stats

	r.Data.LatestConnectPercentiles, r.Data.LatestTotalPercentiles, r.Data.LatestResponsePercentiles = r.GeneratePercentiles(stats)
	r.Data.PercentileTitles = []string{}
	for _, percentile := range r.Data.Latest.Percentiles {
		r.Data.PercentileTitles = append(r.Data.PercentileTitles, fmt.Sprintf("%vth ",percentile * 100))
	}

	r.Data.TotalHistogramTimes = []float64{}
	for _, latency := range r.Data.Latest.TimeToRespond {
		r.Data.TotalHistogramTimes = append(r.Data.TotalHistogramTimes, latency / 1000 / 1000 / 1000)
	}

	r.Data.ConnectionHistogramTimes = []float64{}
	for _, latency := range r.Data.Latest.TimeToConnect {
		r.Data.ConnectionHistogramTimes = append(r.Data.ConnectionHistogramTimes, latency / 1000 / 1000 / 1000)
	}

	r.Data.ProgressBarMax = r.Data.Latest.TotalTestDuration.Nanoseconds()
	r.Data.ProgressBarCurrent = r.Data.Latest.TimeElapsed.Nanoseconds()
	r.Data.PercentageComplete = fmt.Sprintf("%.2f", ( float64(r.Data.ProgressBarCurrent) / float64(r.Data.ProgressBarMax) ) * 100)
}

func (r *RenderHTML) Render() {
	htmlTempl := template.New("testResults")
	templateBytes, err := ioutil.ReadFile("./lib/static/template.html")
	if (err != nil) {
		fmt.Printf("err %v \n",err)
	}
	htmlTempl.Parse(string(templateBytes))

	buf := bytes.NewBufferString("")
	err = htmlTempl.ExecuteTemplate(buf, "testResults", r.Data)
	if (err != nil) {
		fmt.Printf("err %v \n",err)
	}

	outputBytes, err := ioutil.ReadAll(buf)
	if (err != nil) {
		fmt.Printf("err %v \n",err)
	}

	ioutil.WriteFile("./output.html", outputBytes, 0644)
}

func (r *RenderHTML)Quit() {

}

func (r *RenderHTML) GeneratePercentiles(stats AggregatedStats) (connectOutput, totalOutput, responseOutput []float64){
	if (stats.TotalRequests == 0){
		return connectOutput, totalOutput, responseOutput
	}

	for index, _ := range stats.Percentiles {
		connectOutput = append(connectOutput, stats.TimeToConnectPercentiles[index].Seconds() )
		totalOutput = append(totalOutput, stats.TotalTimePercentiles[index].Seconds() )
		responseOutput = append(responseOutput, stats.TimeToRespondPercentiles[index].Seconds() )
	}

	return connectOutput, totalOutput, responseOutput
}
