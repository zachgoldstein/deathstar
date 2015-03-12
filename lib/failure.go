package lib

import "fmt"

func Failure(stats AggregatedStats, harvest float64, yield float64, throughput float64, percentileLatencies []float64) (failure bool, failureDescription string) {
	if (stats.Harvest < harvest) {
		return true, fmt.Sprintf("Harvest of %v is below expected harvest of %v",stats.Harvest, harvest)
	}

	if (stats.Yield < yield) {
		return true, fmt.Sprintf("Yield of %v is below expected yield of %v", stats.Yield, yield)
	}

	if (stats.AverageRespThroughput < throughput) {
		return true, fmt.Sprintf("Throughput of %v resp/s is below expected yield of %v resp/s", stats.AverageRespThroughput, throughput)
	}

	for index, expectedLatency := range percentileLatencies {
		if index < len(stats.TotalTimePercentiles) - 1 {
			totalTimePercentile := stats.TotalTimePercentiles[index].Seconds()
			if (totalTimePercentile > expectedLatency) {
				return true, fmt.Sprintf("%v percentile latency of %v is longer than expected latency of %v", stats.Percentiles[index], totalTimePercentile, expectedLatency )
			}
		}
	}

	return false, ""
}
