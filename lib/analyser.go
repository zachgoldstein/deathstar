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

	TotalRequests int
	AvgConcurrentExecutors int
	MaxConcurrentExecutors int

	Percentiles []float64

	TotalTimePercentiles []time.Duration
	MaxTotalTime time.Duration

	TimeToRespondPercentiles []time.Duration
	MaxTimeToRespond time.Duration

	TimeToConnectPercentiles []time.Duration
	MaxTimeToConnect time.Duration

	Failures int
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
	go a.Analyse()
}

func (a *Analyser) Analyse() {
	for _ = range a.Ticker.C {
		Log("analyse", fmt.Sprintln("Analysing mock") )

		stats := AggregatedStats{
			RawStats : a.Accumulator.Stats,
			Percentiles : a.Percentiles,
		}

		stats.TimeToConnectPercentiles, stats.TimeToRespondPercentiles, stats.TotalTimePercentiles = DeterminePercentilesLatencies(a.Percentiles, a.Accumulator.Stats)

		stats.MaxTotalTime, stats.MaxTimeToRespond, stats.MaxTimeToConnect = DetermineMaxLatencies(a.Accumulator.Stats)

		stats.TotalRequests = len(a.Accumulator.Stats)

		stats.AvgConcurrentExecutors = AverageConcurrency(a.Accumulator.Stats)

		stats.MaxConcurrentExecutors = MaxConcurrency(a.Accumulator.Stats)

		stats.Failures, stats.FailureCounts = GroupFailures(a.Accumulator.Stats)

		stats.TimeToRespond, stats.TimeToConnect, stats.TotalTime = extractLatencies(a.Accumulator.Stats)

		Log("analyse", fmt.Sprintln("Performed analysis and sent to channel ",stats, " ConcurrentExecutors Avg ",stats.AvgConcurrentExecutors, " Max ",stats.MaxConcurrentExecutors) )
		a.StatsChan <- stats
	}
}

func AverageConcurrency(stats []ResponseStats) int {
	if (len(stats) == 0 ) { return 0}
	total := 0
	for _, stat := range stats {
		total += stat.NumExecutors
	}
	return int( math.Ceil( float64(total / len(stats)) ) )
}

func MaxConcurrency(stats []ResponseStats) int {
	if (len(stats) == 0 ) { return 0}
	max := 0
	for _, stat := range stats {
		if (stat.NumExecutors > max) {
			max = stat.NumExecutors
		}
	}
	return max
}

func GroupFailures(stats []ResponseStats) (failures int, failureGroups map[string]int) {
	failureGroups = make(map[string]int)
	for _, stat := range stats {
		if stat.Failure {
			failures += 1
			if fails, ok := failureGroups[stat.FailCategory]; ok {
				failureGroups[stat.FailCategory] = fails + 1
			} else {
				failureGroups[stat.FailCategory] = 1
			}
		}
	}
	Log("analyse", fmt.Sprintln("Grouped ",failures, " failures into map, ",failureGroups) )
	return failures, failureGroups
}

func DetermineMaxLatencies(stats []ResponseStats) (maxTotalTime time.Duration, maxTimeToRespond time.Duration, maxTimeToConnect time.Duration) {
	maxTotalTimeInt := int64(0)
	maxTimeToRespondInt := int64(0)
	maxTimeToConnectInt := int64(0)
	for _, stat := range stats {
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
		TimeToRespond = append(TimeToRespond, float64(stat.TimeToRespond.Nanoseconds()))
		TimeToConnect = append(TimeToConnect, float64(stat.TimeToConnect.Nanoseconds()))
		TotalTime = append(TotalTime, float64(stat.TotalTime.Nanoseconds()))
	}
	return
}
