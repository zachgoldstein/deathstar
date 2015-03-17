package lib

import (
	"time"
	"fmt"
	"sort"
	"math"
	"sync"
)

type Analyser struct {
	Frequency time.Duration
	Ticker *time.Ticker
	ThroughputTicker *time.Ticker
	Accumulator *Accumulator
	StatsChan chan AggregatedStats
	Percentiles []float64

	Harvest float64
	Yield float64
	RespThroughput float64
	PercentilesLatencies []float64

	StartTime time.Time
	WarmUpTime time.Duration

	CalculateRate bool

	Fail chan bool

	mu sync.Mutex
	ThroughputBytes []float64
	ThroughputResps []float64
}

const throughputFrequency = time.Millisecond * 500

type AggregatedStats struct {
	RawStats []ResponseStats
	OverallStats []OverallStats

	StartTime time.Time
	TotalTestDuration time.Duration
	TimeElapsed time.Duration
	TimeWaitingOnFinalReqs time.Duration

	TotalRequests int
	TotalResponses int
	TotalValidResponses int
	AvgConcurrentExecutors int
	MaxConcurrentExecutors int

	Yield float64
	Harvest float64

	LatestByteThroughput float64
	LatestRespThroughput float64
	AverageByteThroughput float64
	AverageRespThroughput float64
	ByteThroughputs []float64
	RespThroughputs []float64

	Percentiles []float64

	TotalTimePercentiles []time.Duration
	MaxTotalTime time.Duration
	MeanTotalTime time.Duration
	MinTotalTime time.Duration

	TimeToRespondPercentiles []time.Duration
	MaxTimeToRespond time.Duration

	TimeToConnectPercentiles []time.Duration
	MaxTimeToConnect time.Duration

	Failures int
	RespFailures int
	ValidationFailures int
	FailureCounts map[string][]DescriptiveError

	TimeToRespond []float64
	TimeToConnect []float64
	TotalTime []float64

	OverallFailure bool
	OverallFailureDescription string

	Rate float64
}

func NewAnalyser(acc *Accumulator, reqOpts RequestOptions, calcRate bool) (*Analyser) {
	analyser := &Analyser{
		Accumulator : acc,
		Frequency : reqOpts.AnalaysisFreqTime,
		StatsChan : make(chan AggregatedStats),
		Percentiles : reqOpts.Percentiles,
		WarmUpTime : reqOpts.WarmUpTime,
		CalculateRate : calcRate,
		Harvest : reqOpts.Harvest,
		Yield : reqOpts.Yield,
		RespThroughput : reqOpts.Throughput,
		PercentilesLatencies : reqOpts.PercentileLatencies,
	}
	analyser.Start()
	return analyser
}

func (a *Analyser) Start() {
	a.StartTime = time.Now()
	a.Ticker = time.NewTicker(a.Frequency)
	a.ThroughputTicker = time.NewTicker(throughputFrequency)
	a.Analyse()
	a.SetupAnalysis()
	Log("analyse", fmt.Sprintln("Analyser started, warmining up for ",a.WarmUpTime, " before performing first analysis") )
}

func (a *Analyser) SetupAnalysis() {
	go func() {
		for _ = range a.Ticker.C {
			if (time.Now().After(a.StartTime.Add(a.WarmUpTime))) {
				a.Analyse()
			}
		}
	}()

	//Calculate throughput (occurs at different rate than overall analysis)
	go func() {
		for _ = range a.ThroughputTicker.C {
			if (time.Now().After(a.StartTime.Add(a.WarmUpTime))) {
				a.SetThroughput()
			}
		}
	}()
}

func (a *Analyser) Stop() {
	a.Ticker.Stop()
	a.ThroughputTicker.Stop()
}

func (a *Analyser) Cleanup() {
	a.SetThroughput()
	a.Analyse()
}

func (a *Analyser) Analyse() {
	if (len(a.Accumulator.OverallStats) == 0 || len(a.Accumulator.Stats) == 0) {
		return
	}

	now := time.Now()

	stats := AggregatedStats{
		Rate : a.Accumulator.OverallStats[len(a.Accumulator.OverallStats) - 1].Rate,
		RawStats : a.Accumulator.Stats,
		OverallStats : a.Accumulator.OverallStats,
		Percentiles : a.Percentiles,
	}

	stats.StartTime, stats.TimeElapsed, stats.TotalTestDuration, stats.TimeWaitingOnFinalReqs = DetermineOverallTimes(stats.OverallStats)

	stats.TimeToConnectPercentiles, stats.TimeToRespondPercentiles, stats.TotalTimePercentiles = DeterminePercentilesLatencies(stats.Percentiles, stats.RawStats)

	stats.MaxTotalTime, stats.MaxTimeToRespond, stats.MaxTimeToConnect = DetermineMaxLatencies(stats.RawStats)

	stats.MinTotalTime = DetermineMinLatencies(stats.RawStats)

	stats.MeanTotalTime = MeanLatencies(stats.RawStats)

	stats.AvgConcurrentExecutors = AverageConcurrency(stats.OverallStats)

	stats.MaxConcurrentExecutors = MaxConcurrency(stats.OverallStats)

	stats.Failures, stats.FailureCounts = GroupFailures(stats.RawStats)

	stats.TimeToRespond, stats.TimeToConnect, stats.TotalTime = extractLatencies(stats.RawStats)

	stats.TotalResponses = NumResponses(stats.RawStats)
	stats.TotalRequests = stats.OverallStats[len(stats.OverallStats) - 1].RequestsIssued

	if (a.CalculateRate) {
		stats.Rate = float64(stats.TotalRequests) / stats.TimeElapsed.Seconds()
	}

	stats.TotalValidResponses = ValidResponses(stats.RawStats)

	stats.Harvest = Harvest(stats.TotalResponses, stats.TotalRequests)
	stats.Yield = Yield(stats.TotalResponses, stats.TotalValidResponses)

	if( len(a.ThroughputBytes) != 0 ) {
		stats.LatestByteThroughput = a.ThroughputBytes[len(a.ThroughputBytes) - 1]
		stats.LatestRespThroughput = a.ThroughputResps[len(a.ThroughputResps) - 1]

		stats.ByteThroughputs = a.ThroughputBytes
		stats.RespThroughputs = a.ThroughputResps

		stats.AverageByteThroughput, stats.AverageRespThroughput = a.AvgThroughput()
	}

	stats.OverallFailure, stats.OverallFailureDescription = Failure(stats, a.Harvest, a.Yield, a.RespThroughput, a.PercentilesLatencies)

	calcTime := time.Since(now)
	Log("analyse", fmt.Sprintln(stats.TotalResponses," valid responses received, ", stats.TotalRequests, " requests issued"))
	Log("analyse", fmt.Sprintln("Sending to stats channel, performed analysis in ",calcTime))

	a.StatsChan <- stats
}

func (a *Analyser) SetThroughput() {
	a.mu.Lock()
	throughputBytes, throughputReqs := a.Throughput(a.Accumulator.Stats)
	a.ThroughputBytes = append(a.ThroughputBytes, throughputBytes)
	a.ThroughputResps = append(a.ThroughputResps, throughputReqs)
	a.mu.Unlock()
}

func (a *Analyser) Throughput(stats []ResponseStats) (byteRate float64, respRate float64) {
	totalBytes := 0
	totalResponses := 0
	now := time.Now()
	lastInterval := now.Add(-throughputFrequency)
	for _, stat := range stats {
		if DoAnalysis(stat) && stat.FinishTime.After(lastInterval) && stat.FinishTime.Before(now) {
			totalBytes += len([]byte(stat.RespPayload))
			totalResponses += 1
		}
	}

	byteRate = float64(totalBytes) / throughputFrequency.Seconds()
	respRate = float64(totalResponses) / throughputFrequency.Seconds()
	return byteRate, respRate
}

func (a *Analyser) AvgThroughput () (avgByteRate float64, avgRespRate float64) {
	if ( len(a.ThroughputBytes) == 0 || len(a.ThroughputResps) == 0) { return }

	totalBytes := 0.0
	for _, byteRate := range a.ThroughputBytes { totalBytes += byteRate }
	avgByteRate = totalBytes / float64( len(a.ThroughputBytes) )

	totalResps := 0.0
	for _, respRate := range a.ThroughputResps { totalResps += respRate }
	avgRespRate = totalResps / float64( len(a.ThroughputResps) )

	return
}

func Harvest(numResponses int, numRequests int) float64 {
	if (numRequests == 0) { return 0.0}
	return float64(numResponses)/float64(numRequests) * 100
}

func ValidResponses(stats []ResponseStats) int {
	validResponses := 0
	for _, stat := range stats {
		if (!stat.Failure()) {
			validResponses += 1
		}
	}
	return validResponses
}

func Yield(numResponses int, validResponses int) float64 {
	if (numResponses == 0) { return 0.0}
	return float64(validResponses)/float64(numResponses) * 100
}

func NumResponses(stats []ResponseStats) int {
	numResponses := 0
	for _, stat := range stats {
		containsResponse := true
		for _, failure := range stat.Failures {
			if _, ok := failure.(RequestExecutionError); ok {
				containsResponse = false
			}
			if _, ok := failure.(StatusCodeError); ok {
				containsResponse = false
			}
		}
		if (containsResponse) {
			numResponses += 1
		}
	}
	return numResponses
}

func DetermineOverallTimes(overallStats []OverallStats) (startTime time.Time, timeElapsed time.Duration, totalTestDuration time.Duration, timeWaitingOnFinalReqs time.Duration)  {
	if (len(overallStats) == 0 ) { return time.Now(), time.Nanosecond, time.Nanosecond, time.Nanosecond}

	latestStat := overallStats[len(overallStats)-1]
	return latestStat.StartTime, latestStat.TimeElapsed, latestStat.TotalTestDuration, latestStat.TimeWaitingOnFinalReqs
}

func AverageConcurrency(stats []OverallStats) int {
	if (len(stats) == 0 ) { return 0}
	total := 0
	for _, stat := range stats {
		total += stat.NumExecutors
	}
	return int( math.Ceil( float64(total / len(stats)) ) )
}

func MaxConcurrency(stats []OverallStats) int {
	if (len(stats) == 0 ) { return 0}
	max := 0
	for _, stat := range stats {
		if (stat.NumExecutors > max) {
			max = stat.NumExecutors
		}
	}
	return max
}

func GroupFailures(stats []ResponseStats) (failures int, failureGroups map[string][]DescriptiveError) {
	failureGroups = make(map[string][]DescriptiveError)
	for _, stat := range stats {
		if stat.Failure() {
			for _, failure := range stat.Failures {
				if fails, ok := failureGroups[failure.Error()]; ok {
					failureGroups[failure.Error()] = append(fails, failure)
				} else {
					failureGroups[failure.Error()] = []DescriptiveError{failure}
				}
			}
			failures += 1
		}
	}
	return
}

func DoAnalysis(stat ResponseStats) bool {
	for _, failure := range stat.Failures {
		if _, ok := failure.(RequestExecutionError); ok {
			return false
		}
		if _, ok := failure.(StatusCodeError); ok {
			return false
		}
	}
	return true
}

func DetermineMaxLatencies(stats []ResponseStats) (maxTotalTime time.Duration, maxTimeToRespond time.Duration, maxTimeToConnect time.Duration) {
	maxTotalTimeInt := int64(0)
	maxTimeToRespondInt := int64(0)
	maxTimeToConnectInt := int64(0)
	for _, stat := range stats {
		if (!DoAnalysis(stat)) { continue }

		if (stat.TotalTime.Nanoseconds() > maxTotalTimeInt) {
			maxTotalTimeInt = stat.TotalTime.Nanoseconds()
		}

		if (stat.TimeToConnect.Nanoseconds() > maxTimeToConnectInt) {
			maxTimeToConnectInt = stat.TimeToConnect.Nanoseconds()
		}

		if (stat.TimeToRespond.Nanoseconds() > maxTimeToRespondInt) {
			maxTimeToRespondInt = stat.TimeToRespond.Nanoseconds()
		}
	}

	maxTotalTime = time.Duration(maxTotalTimeInt) * time.Nanosecond
	maxTimeToConnect = time.Duration(maxTimeToConnectInt) * time.Nanosecond
	maxTimeToRespond = time.Duration(maxTimeToRespondInt) * time.Nanosecond
	return
}

func DetermineMinLatencies(stats []ResponseStats) (minTotalTime time.Duration) {
	minTotalTimeInt := int64(0)
	for _, stat := range stats {
		if (!DoAnalysis(stat)) { continue }

		if (minTotalTimeInt == 0 || stat.TotalTime.Nanoseconds() < minTotalTimeInt) {
			minTotalTimeInt = stat.TotalTime.Nanoseconds()
		}
	}

	minTotalTime = time.Duration(minTotalTimeInt) * time.Nanosecond
	return
}

func MeanLatencies(stats []ResponseStats) (meanTotalTime time.Duration) {
	totalLatency := int64(0)
	numSuccesses := 0
	for _, stat := range stats {
		if (!DoAnalysis(stat)) { continue }

		numSuccesses += 1
		totalLatency += stat.TotalTime.Nanoseconds()
	}
	if (numSuccesses == 0 ) {return time.Second * time.Duration(0)}
	meanLatency := totalLatency / int64( numSuccesses )

	return time.Duration(meanLatency) * time.Nanosecond
}

func DeterminePercentilesLatencies(percentiles []float64, stats []ResponseStats) (TimeToConnectPercentiles, TimeToRespondPercentiles, TotalTimePercentiles []time.Duration) {
	TotalTimes := []int{}
	TimeToResponds := []int{}
	TimeToConnects := []int{}

	for _, stat := range stats {
		if (!DoAnalysis(stat)) { continue }

		TotalTimes = append(TotalTimes, int(stat.TotalTime.Nanoseconds()))
		TimeToResponds = append(TimeToResponds, int(stat.TimeToRespond.Nanoseconds()))
		TimeToConnects = append(TimeToConnects, int(stat.TimeToConnect.Nanoseconds()))
	}

	if len(TotalTimes) == 0 {
		return TimeToConnectPercentiles, TimeToRespondPercentiles, TotalTimePercentiles
	}

	sort.Ints(TotalTimes)
	sort.Ints(TimeToResponds)
	sort.Ints(TimeToConnects)

	TimeToConnectPercentiles = make([]time.Duration, len(percentiles))
	TimeToRespondPercentiles = make([]time.Duration, len(percentiles))
	TotalTimePercentiles = make([]time.Duration, len(percentiles))

	for index, percentile := range percentiles {
		percentileIndexRaw := float64(len(TotalTimes)-1) * percentile
		percentileIndexRaw = math.Ceil(percentileIndexRaw)
		percentileIndex := int(percentileIndexRaw)

		TimeToConnectPercentiles[index] = time.Duration(TimeToConnects[percentileIndex]) * time.Nanosecond
		TimeToRespondPercentiles[index] = time.Duration(TimeToResponds[percentileIndex]) * time.Nanosecond
		TotalTimePercentiles[index] = time.Duration(TotalTimes[percentileIndex]) * time.Nanosecond
	}

	return TimeToConnectPercentiles, TimeToRespondPercentiles, TotalTimePercentiles
}

func extractLatencies(stats []ResponseStats) (TimeToRespond, TimeToConnect, TotalTime []float64) {
	for _, stat := range stats {
		respond := float64( stat.TimeToRespond.Nanoseconds() )
		if (respond != 0) { TimeToRespond = append(TimeToRespond, respond) }

		connect := float64( stat.TimeToConnect.Nanoseconds() )
		if (connect != 0) { TimeToConnect = append(TimeToConnect, connect) }

		total := float64( stat.TotalTime.Nanoseconds() )
		if (total != 0) {TotalTime = append(TotalTime, total ) }
	}
	return
}

func commonFixes() {
	// If "can't assign requested address" errors occur, you're likely bumping into limits imposed by local client OS
}
