package analyse

import (
	"math"
	"sort"
)

type figures struct {
	Name                  string                       `json:"name"`
	Provider              string                       `json:"provider"`
	Seed                  string                       `json:"seed"`
	CountUnhealthyPeriods int                          `json:"countUnhealthyPeriods"`
	CountRequests         int                          `json:"countRequest"`
	CountTimeouts         int                          `json:"countRequestTimeouts"`
	DownPeriods           *figuresDowntimePeriods      `json:"downTimesSec"`
	ResponseTimeDuration  *figuresResponseTimeDuration `json:"responseTimesMs"`

	downPeriodsStore     durationList
	requestDurationStore responseTimeList
}

type figuresResponseTimeDuration struct {
	Min    int     `json:"min"`
	Max    int     `json:"max"`
	Avg    float64 `json:"avg"`
	Median float64 `json:"median"`
	Std    float64 `json:"std"`
}

type figuresDowntimePeriods struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Avg    float64 `json:"avg"`
	Median float64 `json:"median"`
	Std    float64 `json:"std"`
}

func (f *figures) calculateDownPeriodStatistics() {
	if f.CountUnhealthyPeriods < 1 {
		return
	}
	f.DownPeriods = &figuresDowntimePeriods{}
	sort.Sort(f.downPeriodsStore)

	var sum, sumSqrt, avg, variance float64
	for _, o := range f.downPeriodsStore {
		sum += o.Seconds()
	}
	avg = sum / float64(f.CountUnhealthyPeriods)

	// Min, Max and Avg
	f.DownPeriods.Min = f.downPeriodsStore[0].Seconds()
	f.DownPeriods.Max = f.downPeriodsStore[f.CountUnhealthyPeriods-1].Seconds()
	f.DownPeriods.Avg = avg

	// Median
	if f.CountUnhealthyPeriods%2 != 0 {
		f.DownPeriods.Median = f.downPeriodsStore[f.CountUnhealthyPeriods/2].Seconds()
	} else {
		f.DownPeriods.Median = (f.downPeriodsStore[f.CountUnhealthyPeriods/2].Seconds() + f.downPeriodsStore[f.CountUnhealthyPeriods/2-1].Seconds()) / 2
	}

	// Standard Deviation
	for _, o := range f.downPeriodsStore {
		sumSqrt += math.Pow(o.Seconds()-avg, 2)
	}
	variance = sumSqrt / float64(f.CountUnhealthyPeriods)
	f.DownPeriods.Std = math.Sqrt(variance)
}

func (f *figures) calculateResponseTimeStatistics() {
	if f.CountRequests-f.CountTimeouts < 1 {
		return
	}
	f.ResponseTimeDuration = &figuresResponseTimeDuration{}
	sort.Sort(f.requestDurationStore)

	var (
		sum                    int
		sumSqrt, avg, variance float64
		len                    = len(f.requestDurationStore)
	)
	for _, d := range f.requestDurationStore {
		sum += *d
	}
	avg = float64(sum / f.CountRequests)

	// Min, Max, Avg
	f.ResponseTimeDuration.Min = *f.requestDurationStore[0]
	f.ResponseTimeDuration.Max = *f.requestDurationStore[len-1]
	f.ResponseTimeDuration.Avg = avg

	// Median
	if f.CountRequests%2 != 0 {
		f.ResponseTimeDuration.Median = float64(*f.requestDurationStore[len/2])
	} else {
		f.ResponseTimeDuration.Median = float64((*f.requestDurationStore[len/2] + *f.requestDurationStore[len/2-1]) / 2)
	}

	// Standard Deviation
	for _, o := range f.requestDurationStore {
		sumSqrt += math.Pow(float64(*o)-avg, 2)
	}
	variance = sumSqrt / float64(f.CountRequests)
	f.ResponseTimeDuration.Std = math.Sqrt(variance)
}
