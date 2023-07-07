package metrics

import (
	"context"
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	totalCounter   prometheus.Counter
	errorCounter   prometheus.Counter
	successCounter prometheus.Counter
	bp             BalanceProvider
}

type BalanceProvider interface {
	GetBalance(ctx context.Context, addrS string) (*big.Int, error)
}

func (m *Metrics) Wrap(bp BalanceProvider) *Metrics {
	m.bp = bp
	return m
}

func New() *Metrics {
	m := new(Metrics)
	m.totalCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "requests_total",
			Help: "How many requests received",
		},
	)
	m.errorCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "requests_err",
			Help: "How many requests failed",
		},
	)
	m.successCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "requests_ok",
			Help: "How many requests were successful",
		},
	)

	prometheus.MustRegister(m.totalCounter, m.errorCounter, m.successCounter)

	return m
}

func (m *Metrics) GetBalance(ctx context.Context, addrS string) (balance *big.Int, err error) {
	m.totalCounter.Inc()

	defer func() {
		if err != nil {
			m.errorCounter.Inc()
			return
		}
		m.successCounter.Inc()
	}()

	return m.bp.GetBalance(ctx, addrS)
}
