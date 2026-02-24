package tok

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const scopeName = "github.com/quexer/tok"

type instruments struct {
	tracer       trace.Tracer
	connOnline   metric.Int64UpDownCounter
	messagesUp   metric.Int64Counter
	messagesDown metric.Int64Counter
	queueEnq     metric.Int64Counter
	queueDeq     metric.Int64Counter
	sendDuration metric.Float64Histogram
}

func newInstruments(mp metric.MeterProvider, tp trace.TracerProvider) *instruments {
	if mp == nil {
		mp = otel.GetMeterProvider()
	}
	if tp == nil {
		tp = otel.GetTracerProvider()
	}

	m := mp.Meter(scopeName)

	connOnline, _ := m.Int64UpDownCounter("tok.connections.online",
		metric.WithDescription("Current number of online connections"),
		metric.WithUnit("{connection}"))

	messagesUp, _ := m.Int64Counter("tok.messages.up",
		metric.WithDescription("Total messages received from clients"),
		metric.WithUnit("{message}"))

	messagesDown, _ := m.Int64Counter("tok.messages.down",
		metric.WithDescription("Total messages sent to clients"),
		metric.WithUnit("{message}"))

	queueEnq, _ := m.Int64Counter("tok.queue.enqueue",
		metric.WithDescription("Total messages enqueued to offline queue"),
		metric.WithUnit("{message}"))

	queueDeq, _ := m.Int64Counter("tok.queue.dequeue",
		metric.WithDescription("Total messages dequeued from offline queue"),
		metric.WithUnit("{message}"))

	sendDuration, _ := m.Float64Histogram("tok.send.duration",
		metric.WithDescription("Duration of Hub.Send operations"),
		metric.WithUnit("s"))

	return &instruments{
		tracer:       tp.Tracer(scopeName),
		connOnline:   connOnline,
		messagesUp:   messagesUp,
		messagesDown: messagesDown,
		queueEnq:     queueEnq,
		queueDeq:     queueDeq,
		sendDuration: sendDuration,
	}
}
