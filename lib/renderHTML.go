package lib

import (
	"fmt"
	"net/http"
	"time"
	socketio     "github.com/googollee/go-socket.io"
	"encoding/json"
	"math"
)

type RenderHTML struct {
	Port int
	Done chan bool
	Data RenderData

	DataSendFrequency time.Duration
	DataSendTicker *time.Ticker
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

	SampledRespThroughputs []float64
	RespThroughPutSampling float64
	SampledByteThroughputs []float64
	ByteThroughPutSampling float64

	SampledConnectionLatencies []float64
	ConnectionLatencySampling float64
	SampledResponseLatencies []float64
	ResponseLatencySampling float64

	ProgressBarMax int64
	ProgressBarCurrent int64
	PercentageComplete string

	ReqProgressBarMax int
	ReqProgressBarCurrent int
	ReqBarText string
	ReqPercentageComplete string

	MaxResponseTime string
	AvgResponseTime string
	TopPercentileTime string
	TopPercentileTimeTitle string
	MinResponseTime string

	TimeElapsed string
	TotalTime string

	Yield string
	Harvest string

	AvgThroughputKbs string
	AvgThroughputResps string

	FailureMap map[string]int
}

func NewRenderHTML(reqOpts RequestOptions) *RenderHTML {
	return &RenderHTML{
		DataSendTicker : time.NewTicker(reqOpts.RenderFrequency),
		Data : RenderData{
			ReqOpts : reqOpts,
		},
	}
}

func (r *RenderHTML) Setup(done chan bool) {
	r.Done = done
	go r.StartSocketServer()
}

func (r *RenderHTML) Generate(stats AggregatedStats) {
	if (stats.TotalRequests == 0 ){
		return
	}

	r.Data.Title = "Testman"
	r.Data.Latest = stats

	if (r.Data.ReqOpts.IncreaseRateToFailure) {
		r.Data.ModeDesc = "Rate is increasing until failure"
	} else if (r.Data.ReqOpts.ExecuteSingleRequest){
		r.Data.ModeDesc = "A single request is being executed"
	} else {
		r.Data.ModeDesc = "Executing as many req/s as possible"
	}

	r.Data.LatestConnectPercentiles, r.Data.LatestTotalPercentiles, r.Data.LatestResponsePercentiles = r.GeneratePercentiles(stats)
	r.Data.PercentileTitles = []string{}
	for _, percentile := range r.Data.Latest.Percentiles {
		r.Data.PercentileTitles = append(r.Data.PercentileTitles, fmt.Sprintf("%vth ",percentile * 100))
	}

	r.Data.ProgressBarMax = r.Data.Latest.TotalTestDuration.Nanoseconds()
	r.Data.ProgressBarCurrent = r.Data.Latest.TimeElapsed.Nanoseconds()
	r.Data.PercentageComplete = fmt.Sprintf("%.2f", ( float64(r.Data.ProgressBarCurrent) / float64(r.Data.ProgressBarMax) ) * 100)

	r.Data.ReqProgressBarMax = r.Data.ReqOpts.RequestsToIssue
	r.Data.ReqProgressBarCurrent = r.Data.Latest.TotalResponses
	r.Data.ReqPercentageComplete = fmt.Sprintf("%.2f", ( float64(r.Data.ReqProgressBarCurrent) / float64(r.Data.ReqProgressBarMax) ) * 100)
	r.Data.ReqBarText = fmt.Sprintf("%v of %v Reqs Executed", r.Data.ReqProgressBarCurrent, r.Data.ReqProgressBarMax)


	r.Data.Yield = fmt.Sprintf("%.2f", r.Data.Latest.Yield)
	r.Data.Harvest = fmt.Sprintf("%.2f", r.Data.Latest.Harvest)

	r.Data.MaxResponseTime = fmt.Sprintf("%.4f", r.Data.Latest.MaxTotalTime.Seconds())
	r.Data.MinResponseTime = fmt.Sprintf("%.4f", r.Data.Latest.MinTotalTime.Seconds())
	r.Data.AvgResponseTime = fmt.Sprintf("%.4f", r.Data.Latest.MeanTotalTime.Seconds())

	if (len(r.Data.PercentileTitles) > 0) {
		r.Data.TopPercentileTimeTitle = r.Data.PercentileTitles[len(r.Data.PercentileTitles) - 1]
	}
	if (len(r.Data.LatestTotalPercentiles) > 0) {
		r.Data.TopPercentileTime = fmt.Sprintf("%.4f", r.Data.LatestTotalPercentiles[len(r.Data.LatestTotalPercentiles) - 1])
	}

	r.Data.AvgThroughputKbs = fmt.Sprintf("%.4f", r.Data.Latest.AverageByteThroughput / 1000.0)
	r.Data.AvgThroughputResps = fmt.Sprintf("%.4f", r.Data.Latest.AverageRespThroughput)

	r.Data.TimeElapsed = r.Data.Latest.TimeElapsed.String()
	r.Data.TotalTime = r.Data.Latest.TotalTestDuration.String()
	r.Data.FailureMap = make(map[string]int)
	for _, failures := range r.Data.Latest.FailureCounts {
		if (len(failures) > 1) {
			r.Data.FailureMap[failures[0].Error()] = len(failures)
		}
	}

	r.Data.SampledRespThroughputs, r.Data.RespThroughPutSampling = r.SampleData(r.Data.Latest.RespThroughputs)
	r.Data.SampledByteThroughputs, r.Data.ByteThroughPutSampling = r.SampleData(r.Data.Latest.ByteThroughputs)

	rawRespondTimesSecs := []float64{}
	for _, latency := range r.Data.Latest.TimeToRespond {
		rawRespondTimesSecs = append(rawRespondTimesSecs, latency / 1000 / 1000 / 1000)
	}
	r.Data.SampledConnectionLatencies, r.Data.ConnectionLatencySampling = r.SampleData(rawRespondTimesSecs)

	rawConnectTimesSecs := []float64{}
	for _, latency := range r.Data.Latest.TimeToConnect {
		rawConnectTimesSecs = append(rawConnectTimesSecs, latency / 1000 / 1000 / 1000)
	}
	r.Data.SampledResponseLatencies, r.Data.ResponseLatencySampling = r.SampleData(rawConnectTimesSecs)

}

const MAX_DATA_SIZE = 250.0

func (r *RenderHTML) SampleData(data []float64) (sampledData []float64, sampling float64) {
	if (float64( len(data) ) < MAX_DATA_SIZE) {
		return data, 1
	}

	sampling = math.Ceil( float64 ( len(data) ) / MAX_DATA_SIZE )

	for index, item := range data {
		if (index % int(sampling) == 0) {
			sampledData = append(sampledData, item)
		}
	}
	return sampledData, sampling
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

func (r *RenderHTML) Render() {
//	htmlTempl := template.New("testResults")
//	templateBytes, err := ioutil.ReadFile("./lib/static/template.html")
//	if (err != nil) {
//		fmt.Printf("err %v \n",err)
//	}
//	htmlTempl.Parse(string(templateBytes))
//
//	buf := bytes.NewBufferString("")
//	err = htmlTempl.ExecuteTemplate(buf, "testResults", r.Data)
//	if (err != nil) {
//		fmt.Printf("err %v \n",err)
//	}
//
//	outputBytes, err := ioutil.ReadAll(buf)
//	if (err != nil) {
//		fmt.Printf("err %v \n",err)
//	}
//
//	ioutil.WriteFile("./output.html", outputBytes, 0644)
//
//
//	Log("reporter","HTML output rendered")
}

func (re *RenderHTML)frontendClient (so socketio.Socket) {

	Log("all", "Received connection message")
	err := so.Join("data")
	err = so.Emit("event","testing???")
	if (err != nil) {
		Log("all", "Error occurred joining data room")
	}
	//Send data periodically to frontend
	go func(so socketio.Socket){
		for _ = range re.DataSendTicker.C{
			data, err := json.Marshal(re.Data)
			if err != nil {
				Log("all", "could not marshall data? ",err.Error())
				return
			}

			err = so.BroadcastTo("data","event",string(data))
			if err != nil {
				Log("all", "could not broadcast message? ",err.Error())
				return
			}
		}
	}(so)


}

func (r *RenderHTML)StartSocketServer() (err error) {
	server, err := socketio.NewServer(nil)
	if err != nil {
		return err
	}
	server.On("connection", r.frontendClient)
	server.On("error", func(so socketio.Socket, err error) {
		Log("all","An error occurred serving a frontend socket ", err.Error())
	})

	http.HandleFunc("/socket.io/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "null")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		server.ServeHTTP(w, r)
	})

	Log("all","Serving frontend socket connections at localhost:8081...")
	err = http.ListenAndServe(":8081", nil)
	if err != nil {
		return err
	}
	return
}

func (r *RenderHTML)Quit() {

}
