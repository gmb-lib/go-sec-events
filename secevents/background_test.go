package secevents_test

import (
	"context"
	"testing"

	"azugo.io/azugo"
	"github.com/go-quicktest/qt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/gmb-lib/go-platform-kit/broker"
	"github.com/gmb-lib/go-sec-events/secevents"
)

// requestOnlySink implements Sink and nothing else — the shape every service's test
// double has, and the reason the background half is a separate interface rather than
// a second method on Sink.
type requestOnlySink struct{}

func (requestOnlySink) Emit(_ *azugo.Context, _ *broker.Envelope) error { return nil }

// sweepEvent is the shape a scheduled task emits: no actor, no resource, a delete.
func sweepEvent() *broker.Envelope {
	return &broker.Envelope{
		EventType:  "document.retention_swept",
		Operation:  broker.OpDelete,
		Outcome:    broker.OutcomeSuccess,
		Attributes: map[string]any{secevents.AttrSeverity: string(secevents.SeverityInfo), "erased": 4},
	}
}

func observed() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.InfoLevel)

	return zap.New(core), logs
}

// A scheduled task can emit without a request, and the line it writes is the one a
// SIEM already selects on.
func TestEmitBackgroundWritesTheSinkLine(t *testing.T) {
	log, logs := observed()
	em := secevents.NewEmitter(secevents.NewLogSinkFor(log))

	qt.Assert(t, qt.IsNil(em.EmitBackground(context.Background(), sweepEvent())))

	entries := logs.FilterMessage("security_event").All()
	qt.Assert(t, qt.Equals(len(entries), 1))

	got := map[string]bool{}
	for _, f := range entries[0].Context {
		got[f.Key] = true
	}
	for _, want := range []string{"event_id", "occurred_at", "event_type", "category", "outcome", "severity", "attributes"} {
		qt.Assert(t, qt.IsTrue(got[want]))
	}
}

// The event is stamped: an id and an occurrence time, so it validates and so the
// SIEM can order and deduplicate it. Without a request there is no correlation id to
// take, and that is the only difference from the request path.
func TestEmitBackgroundStampsTheEvent(t *testing.T) {
	log, logs := observed()
	em := secevents.NewEmitter(secevents.NewLogSinkFor(log))

	qt.Assert(t, qt.IsNil(em.EmitBackground(context.Background(), sweepEvent())))

	entry := logs.FilterMessage("security_event").All()[0]
	var id string
	var stamped bool
	for _, f := range entry.Context {
		if f.Key == "event_id" {
			id = f.String
		}
		if f.Key == "occurred_at" {
			stamped = true
		}
	}
	qt.Assert(t, qt.Not(qt.Equals(id, "")))
	qt.Assert(t, qt.IsTrue(stamped))
}

// The act's kind reaches the log. Without it a deletion is indistinguishable from
// any other event on this path, and a SIEM rule has to infer it from the event type.
func TestEmitBackgroundRecordsTheOperation(t *testing.T) {
	log, logs := observed()
	em := secevents.NewEmitter(secevents.NewLogSinkFor(log))

	qt.Assert(t, qt.IsNil(em.EmitBackground(context.Background(), sweepEvent())))

	var op string
	for _, f := range logs.FilterMessage("security_event").All()[0].Context {
		if f.Key == "operation" {
			op = f.String
		}
	}
	qt.Assert(t, qt.Equals(op, string(broker.OpDelete)))
}

// Severity still picks the level, so SIEM alerting works on the background path too.
func TestEmitBackgroundMapsSeverityToLevel(t *testing.T) {
	log, logs := observed()
	em := secevents.NewEmitter(secevents.NewLogSinkFor(log))

	ev := sweepEvent()
	ev.Attributes[secevents.AttrSeverity] = string(secevents.SeverityHigh)
	qt.Assert(t, qt.IsNil(em.EmitBackground(context.Background(), ev)))

	qt.Assert(t, qt.Equals(logs.FilterLevelExact(zapcore.ErrorLevel).Len(), 1))
}

// Attributes are stripped on this path exactly as on the request path — background
// work is no less able to put a token in a map by accident.
func TestEmitBackgroundSanitizesAttributes(t *testing.T) {
	log, logs := observed()
	em := secevents.NewEmitter(secevents.NewLogSinkFor(log))

	ev := sweepEvent()
	ev.Attributes["authorization"] = "Bearer something"
	qt.Assert(t, qt.IsNil(em.EmitBackground(context.Background(), ev)))

	for _, f := range logs.FilterMessage("security_event").All()[0].Context {
		if f.Key != "attributes" {
			continue
		}
		attrs, ok := f.Interface.(map[string]any)
		qt.Assert(t, qt.IsTrue(ok))
		_, present := attrs["authorization"]
		qt.Assert(t, qt.IsFalse(present))
	}
}

// An invalid event is refused rather than written, the same as on the request path.
func TestEmitBackgroundValidates(t *testing.T) {
	log, logs := observed()
	em := secevents.NewEmitter(secevents.NewLogSinkFor(log))

	// No event type: nothing downstream could route it.
	err := em.EmitBackground(context.Background(), &broker.Envelope{Outcome: broker.OutcomeSuccess})
	qt.Assert(t, qt.IsNotNil(err))
	qt.Assert(t, qt.Equals(logs.FilterMessage("security_event").Len(), 0))
}

// A log sink built for the request path only has no logger of its own, so it says
// so instead of panicking or dropping the event on the floor.
func TestEmitBackgroundRefusesARequestOnlyLogSink(t *testing.T) {
	em := secevents.NewEmitter(secevents.NewLogSink())

	err := em.EmitBackground(context.Background(), sweepEvent())
	qt.Assert(t, qt.IsNotNil(err))
}

// A sink that implements Sink alone cannot serve background callers, and is told so
// plainly. This is the case that must never become a silent drop: it is how an
// erasure record would go missing.
func TestEmitBackgroundRefusesASinkWithoutTheBackgroundHalf(t *testing.T) {
	err := secevents.NewEmitter(requestOnlySink{}).EmitBackground(context.Background(), sweepEvent())
	qt.Assert(t, qt.IsNotNil(err))
}

// The broker sink really publishes from a plain context. This is the half that was
// broken before: a deployment on the broker sink got request events on the broker
// and background events nowhere near it.
func TestEmitBackgroundPublishesThroughTheBrokerSink(t *testing.T) {
	tr := &captureTransport{}
	em := secevents.NewEmitter(secevents.NewBrokerSink(broker.NewPublisher(tr, "svc"), secevents.DefaultTopic))

	qt.Assert(t, qt.IsNil(em.EmitBackground(context.Background(), sweepEvent())))

	got := tr.last()
	qt.Assert(t, qt.IsNotNil(got))
	qt.Assert(t, qt.Equals(got.EventType, "document.retention_swept"))
	qt.Assert(t, qt.Not(qt.Equals(got.EventID, "")))
	qt.Assert(t, qt.IsFalse(got.OccurredAt.IsZero()))
}

// A nil emitter, a nil sink and a nil envelope are all answered rather than fatal:
// telemetry must never be what takes a sweep down.
func TestEmitBackgroundIsSafeOnNilInputs(t *testing.T) {
	var nilEmitter *secevents.Emitter
	qt.Assert(t, qt.IsNotNil(nilEmitter.EmitBackground(context.Background(), sweepEvent())))

	qt.Assert(t, qt.IsNotNil(secevents.NewEmitter(nil).EmitBackground(context.Background(), sweepEvent())))

	log, _ := observed()
	em := secevents.NewEmitter(secevents.NewLogSinkFor(log))
	qt.Assert(t, qt.IsNotNil(em.EmitBackground(context.Background(), nil)))
}
