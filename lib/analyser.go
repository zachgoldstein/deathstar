package lib

import (
	"time"
	"fmt"
	"sort"
	"math"
)

//Analyser will
type Analyser struct {
	Frequency time.Duration
	Ticker *time.Ticker
	Accumulator *Accumulator
	StatsChan chan AggregatedStats
	Percentiles []float64
}

type AggregatedStats struct {
	RawStats []ResponseStats

	StartTime time.Time
	TotalTestDuration time.Duration
	TimeElapsed time.Duration

	TotalRequests int
	TotalResponses int
	TotalValidResponses int
	AvgConcurrentExecutors int
	MaxConcurrentExecutors int

	Yield float64
	Harvest float64

	//TODO: add a measure of the actual req/s the system is executing, avg, max, min

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
	FailureCounts map[string]int

	TimeToRespond []float64
	TimeToConnect []float64
	TotalTime []float64
}

func NewAnalyser(acc *Accumulator, frequency time.Duration, percentiles []float64) (*Analyser) {
	analyser := &Analyser{
		Accumulator : acc,
		Frequency : frequency,
		StatsChan : make(chan AggregatedStats),
		Percentiles : percentiles,
	}
	analyser.Start()
	return analyser
}

func (a *Analyser) Start() {
	a.Ticker = time.NewTicker(a.Frequency)
	a.Analyse()
}

func (a *Analyser) Analyse() {
	go func() {
		for _ = range a.Ticker.C {

			Log("analyse", fmt.Sprintln("Analysing mock") )

			stats := AggregatedStats{
				RawStats : a.Accumulator.Stats,
				Percentiles : a.Percentiles,
			}

			stats.StartTime, stats.TimeElapsed, stats.TotalTestDuration = DetermineOverallTimes(a.Accumulator.OverallStats)

			stats.TimeToConnectPercentiles, stats.TimeToRespondPercentiles, stats.TotalTimePercentiles = DeterminePercentilesLatencies(a.Percentiles, a.Accumulator.Stats)

			stats.MaxTotalTime, stats.MaxTimeToRespond, stats.MaxTimeToConnect = DetermineMaxLatencies(a.Accumulator.Stats)

			stats.MinTotalTime = DetermineMinLatencies(a.Accumulator.Stats)

			stats.MeanTotalTime = MeanLatencies(a.Accumulator.Stats)

			stats.AvgConcurrentExecutors = AverageConcurrency(a.Accumulator.OverallStats)

			stats.MaxConcurrentExecutors = MaxConcurrency(a.Accumulator.OverallStats)

			stats.Failures, stats.RespFailures, stats.ValidationFailures, stats.FailureCounts = GroupFailures(a.Accumulator.Stats)

			stats.TimeToRespond, stats.TimeToConnect, stats.TotalTime = extractLatencies(a.Accumulator.Stats)

			stats.TotalResponses = NumResponses(a.Accumulator.Stats)
			stats.TotalRequests = a.Accumulator.OverallStats[len(a.Accumulator.OverallStats) - 1].NumRequests

			stats.TotalValidResponses = ValidResponses(a.Accumulator.Stats)

			stats.Harvest = Harvest(stats.TotalResponses, stats.TotalRequests)
			stats.Yield = Yield(stats.TotalResponses, stats.TotalValidResponses)

			Log("analyse", fmt.Sprintln("Performed analysis and sent to channel ",stats, " ConcurrentExecutors Avg ",stats.AvgConcurrentExecutors, " Max ",stats.MaxConcurrentExecutors) )
			a.StatsChan <- stats
		}
	}()
}

func Harvest(numResponses int, numRequests int) float64 {
	if (numRequests == 0) { return 0.0}
	return float64(numResponses)/float64(numRequests) * 100
}

func ValidResponses(stats []ResponseStats) int {
	validResponses := 0
	for _, stat := range stats {
		if (!stat.ValidationErr && !stat.Failure) {
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
	numRequests := len(stats)
	if (numRequests == 0) { return 0}
	numResponses := 0
	for _, stat := range stats {
		if (!stat.RespErr) {
			numResponses += 1
		}
	}
	return numResponses
}

func DetermineOverallTimes(overallStats []OverallStats) (startTime time.Time, timeElapsed time.Duration, totalTestDuration time.Duration)  {
	if (len(overallStats) == 0 ) { return time.Now(), time.Nanosecond, time.Nanosecond}

	latestStat := overallStats[len(overallStats)-1]
	return latestStat.StartTime, latestStat.TimeElapsed, latestStat.TotalTestDuration
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

func GroupFailures(stats []ResponseStats) (failures int, respErrs int, validationErrs int, failureGroups map[string]int) {
	failureGroups = make(map[string]int)
	for _, stat := range stats {
		if stat.Failure {
			if stat.RespErr { respErrs += 1}
			if stat.ValidationErr { validationErrs += 1}

			failures += 1
			if fails, ok := failureGroups[stat.FailCategory]; ok {
				failureGroups[stat.FailCategory] = fails + 1
			} else {
				failureGroups[stat.FailCategory] = 1
			}
		}
	}
	Log("analyse", fmt.Sprintln("Grouped ",failures, " failures into map, ",failureGroups) )
	return
}

func DetermineMaxLatencies(stats []ResponseStats) (maxTotalTime time.Duration, maxTimeToRespond time.Duration, maxTimeToConnect time.Duration) {
	maxTotalTimeInt := int64(0)
	maxTimeToRespondInt := int64(0)
	maxTimeToConnectInt := int64(0)
	for _, stat := range stats {
		if (stat.RespErr) { continue }

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
		if (stat.RespErr) { continue }
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
		if (stat.RespErr) { continue }
		numSuccesses += 1
		totalLatency += stat.TotalTime.Nanoseconds()
	}
	if (numSuccesses == 0 ) {return time.Second * time.Duration(0)}
	meanLatency := totalLatency / int64( numSuccesses )

	return time.Duration(meanLatency) * time.Nanosecond
}

func DeterminePercentilesLatencies(percentiles []float64, stats []ResponseStats) (TimeToConnectPercentiles, TimeToRespondPercentiles, TotalTimePercentiles []time.Duration) {
	if len(stats) == 0 {
		return TimeToConnectPercentiles, TimeToRespondPercentiles, TotalTimePercentiles
	}

	TotalTimes := []int{}
	TimeToResponds := []int{}
	TimeToConnects := []int{}

	for _, stat := range stats {
		TotalTimes = append(TotalTimes, int(stat.TotalTime.Nanoseconds()))
		TimeToResponds = append(TimeToResponds, int(stat.TimeToRespond.Nanoseconds()))
		TimeToConnects = append(TimeToConnects, int(stat.TimeToConnect.Nanoseconds()))
	}

	sort.Ints(TotalTimes)
	sort.Ints(TimeToResponds)
	sort.Ints(TimeToConnects)

	TimeToConnectPercentiles = make([]time.Duration, len(percentiles))
	TimeToRespondPercentiles = make([]time.Duration, len(percentiles))
	TotalTimePercentiles = make([]time.Duration, len(percentiles))

	for index, percentile := range percentiles {
		percentileIndexRaw := float64(len(stats)-1) * percentile
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
