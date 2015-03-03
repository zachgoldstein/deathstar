package lib

import (
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
	ReqOpts RequestOptions

	Latest AggregatedStats
	Title string

	ModeDesc string

	LatestConnectPercentiles []float64
 	LatestTotalPercentiles []float64
 	LatestResponsePercentiles []float64
	PercentileTitles []string

	TotalHistogramTimes []float64
	ConnectionHistogramTimes []float64

	ProgressBarMax int64
	ProgressBarCurrent int64
	PercentageComplete string

	MaxResponseTime string
	AvgResponseTime string
	TopPercentileTime string
	TopPercentileTimeTitle string
	MinResponseTime string

	Yield string
	Harvest string

	ThroughputKbs float64
	AvgThroughputKbs string
	AvgThroughputResps string
}

func NewRenderHTML(reqOpts RequestOptions) *RenderHTML {
	return &RenderHTML{
		Data : RenderData{
			ReqOpts : reqOpts,
		},
	}
}

func (r *RenderHTML) Setup(done chan bool) {
	r.Done = done
}

func (r *RenderHTML) Generate(stats AggregatedStats) {
	if (stats.TotalRequests == 0 ){
		return
	}

	r.Data.Title = "Testman"
	r.Data.Latest = stats

	if (r.Data.ReqOpts.ReqLimitMode == "total") {
		r.Data.ModeDesc = "Executing all requests"
	} else if (r.Data.ReqOpts.ReqLimitMode == "rate"){
		r.Data.ModeDesc = "Executing at specified rate"
	}

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

	r.Data.Yield = fmt.Sprintf("%.2f", r.Data.Latest.Yield)
	r.Data.Harvest = fmt.Sprintf("%.2f", r.Data.Latest.Harvest)

	r.Data.MaxResponseTime = fmt.Sprintf("%.4f", r.Data.Latest.MaxTotalTime.Seconds())
	r.Data.MinResponseTime = fmt.Sprintf("%.4f", r.Data.Latest.MinTotalTime.Seconds())
	r.Data.AvgResponseTime = fmt.Sprintf("%.4f", r.Data.Latest.MeanTotalTime.Seconds())
	r.Data.TopPercentileTimeTitle = r.Data.PercentileTitles[len(r.Data.PercentileTitles) - 1]
	r.Data.TopPercentileTime = fmt.Sprintf("%.4f", r.Data.LatestTotalPercentiles[len(r.Data.LatestTotalPercentiles) - 1])

	r.Data.ThroughputKbs = r.Data.Latest.LatestByteThroughput / 1000.0
	r.Data.AvgThroughputKbs = fmt.Sprintf("%.4f", r.Data.Latest.AverageByteThroughput / 1000.0)
	r.Data.AvgThroughputResps = fmt.Sprintf("%.4f", r.Data.Latest.AverageRespThroughput)
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


	Log("reporter","HTML output rendered")
}

func (r *RenderHTML)Quit() {

}

func (r *RenderHTML) GeneratePercentiles(stats AggregatedStats) (connectOutput, totalOutput, responseOutput []float64){
	if (stats.TotalRequests == 0 ){
		return connectOutput, totalOutput, responseOutput
	}
	for index, _ := range stats.Percentiles {

		//TODO: Hack to deal with issue in percentile generation.... fix that.
		if (len(stats.TimeToConnectPercentiles) -1 < index ||
			len(stats.TotalTimePercentiles) -1 < index ||
			len(stats.TimeToRespondPercentiles) -1 < index ) {
			break
		}

		connectOutput = append(connectOutput, stats.TimeToConnectPercentiles[index].Seconds() )
		totalOutput = append(totalOutput, stats.TotalTimePercentiles[index].Seconds() )
		responseOutput = append(responseOutput, stats.TimeToRespondPercentiles[index].Seconds() )
	}

	return connectOutput, totalOutput, responseOutput
}
