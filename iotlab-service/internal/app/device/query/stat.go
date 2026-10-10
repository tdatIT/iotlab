// Package query holds the device read use cases.
package query

import "github.com/tdatIT/iotlab-service/internal/domain/dtos/devicedtos"

// dayWindow is the window of the 24h endpoints.
const dayWindow = "24h"

// accumulator folds values, or pre-aggregated buckets, into one min/max/avg Stat.
type accumulator struct {
	min, max, sum float64
	n             int64
}

func (a *accumulator) add(v float64) { a.addBucket(v, v, v, 1) }

// addBucket folds a bucket of n samples; its avg is weighted by n so the result equals the average of the raw samples.
func (a *accumulator) addBucket(lo, hi, avg float64, n int64) {
	if n == 0 {
		return
	}
	if a.n == 0 || lo < a.min {
		a.min = lo
	}
	if a.n == 0 || hi > a.max {
		a.max = hi
	}
	a.sum += avg * float64(n)
	a.n += n
}

// stat returns nil when nothing was folded.
func (a *accumulator) stat() *devicedtos.Stat {
	if a.n == 0 {
		return nil
	}
	return &devicedtos.Stat{Min: a.min, Max: a.max, Avg: a.sum / float64(a.n)}
}

// avg returns nil when nothing was folded.
func (a *accumulator) avg() *float64 {
	if a.n == 0 {
		return nil
	}
	return new(a.sum / float64(a.n))
}
