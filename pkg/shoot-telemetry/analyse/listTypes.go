package analyse

import "time"

type durationList []time.Duration

func (d durationList) Less(i, j int) bool {
	return d[i].Seconds() < d[j].Seconds()
}

func (d durationList) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d durationList) Len() int {
	return len(d)
}

type responseTimeList []*int

func (d responseTimeList) Less(i, j int) bool {
	return *d[i] < *d[j]
}

func (d responseTimeList) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (d responseTimeList) Len() int {
	return len(d)
}
