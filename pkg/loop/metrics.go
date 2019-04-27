package loop

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	totalJobs = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "jobs_total",
		Help: "total number of jobs",
	}, []string{"loop", "queue"})

	queuedJobs = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "jobs_queued",
		Help: "Current number of queued jobs",
	}, []string{"loop", "queue"})

	jobExecDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "job_exec_duration",
			Help:    "Histogram for the job execution duration of the event loop",
			Buckets: prometheus.ExponentialBuckets(0.1, 1.5, 5),
		},
		[]string{"loop"},
	)
)

func init() {
	prometheus.MustRegister(totalJobs, queuedJobs, jobExecDuration)
}
