package secevents

import (
	"context"
	"errors"

	"azugo.io/azugo"
	"go.uber.org/zap"

	"github.com/gmb-lib/go-platform-kit/broker"
)

// DefaultTopic is the broker topic used by BrokerSink for security events.
const DefaultTopic = "audit.security"

// logMessage is the fixed message every security log line carries, so the SIEM
// can select the stream by message and index on the structured fields.
const logMessage = "security_event"

// BackgroundSink is the optional second half of Sink: a sink that can also deliver
// an event from work with no request behind it — a scheduled sweep, a purge, a
// drainer. It is a separate interface rather than a second method on Sink so that
// adding it breaks no existing implementation, including the test doubles services
// keep; a sink that does not implement it simply cannot serve background callers,
// and Emitter.EmitBackground says so rather than failing obscurely.
type BackgroundSink interface {
	EmitBackground(ctx context.Context, ev *broker.Envelope) error
}

// LogSink emits security events as structured log lines. On the request path it
// writes to the request logger, so every line inherits that request's correlation
// and trace ids; the platform's log pipeline ships them to the SIEM / central log
// management — the common NIS2-audit path. Severity maps to the log level so SIEM
// alerting and dashboards work without parsing the payload.
//
// Background work has no request and therefore no logger to borrow, so a LogSink
// that must serve it is built with one of its own (NewLogSinkFor).
type LogSink struct {
	log *zap.Logger
}

// NewLogSink returns a LogSink for the request path only. It borrows the request's
// logger on every write, so it has none of its own and cannot serve background
// callers; use NewLogSinkFor where the service also emits from scheduled work.
func NewLogSink() *LogSink { return &LogSink{} }

// NewLogSinkFor returns a LogSink that can serve both paths: request events still
// go to the request logger, and background events go to log. Pass the service's own
// logger — the one the application was built with.
func NewLogSinkFor(log *zap.Logger) *LogSink { return &LogSink{log: log} }

// Emit writes ev to the request logger. The correlation_id/trace_id are already
// on the logger (correlation middleware); this adds the event-specific fields.
func (s *LogSink) Emit(ctx *azugo.Context, ev *broker.Envelope) error {
	if ev == nil {
		return errors.New("secevents: nil envelope")
	}

	return write(ctx.Log(), ev)
}

// EmitBackground writes ev to the sink's own logger, for a caller with no request.
// The line carries no correlation or trace id because there is no request to take
// them from — everything else about it is identical to the request path, because
// both go through the same field builder. A SIEM selects on the message and the
// field names, so a divergence between the two would be invisible in test and total
// in production.
func (s *LogSink) EmitBackground(_ context.Context, ev *broker.Envelope) error {
	if ev == nil {
		return errors.New("secevents: nil envelope")
	}
	if s == nil || s.log == nil {
		return errors.New("secevents: log sink has no logger for background events — build it with NewLogSinkFor")
	}

	return write(s.log, ev)
}

// write renders ev onto log at the level its severity maps to. The single place the
// line's shape is decided, so the request and background paths cannot drift apart.
func write(log *zap.Logger, ev *broker.Envelope) error {
	fields := make([]zap.Field, 0, 12)
	fields = append(fields,
		zap.String("event_id", ev.EventID),
		zap.Time("occurred_at", ev.OccurredAt),
		zap.String("event_type", ev.EventType),
		zap.String("category", string(broker.CategorySecurity)),
		zap.String("outcome", string(ev.Outcome)),
		zap.String(AttrSeverity, string(severityOf(ev))),
	)

	// What kind of act it was. Without it a deletion reads like any other event on
	// the log path, so a SIEM rule has to infer it from the event type alone.
	if ev.Operation != "" {
		fields = append(fields, zap.String("operation", string(ev.Operation)))
	}

	if ev.Actor != nil {
		fields = append(fields,
			zap.String("actor_id", ev.Actor.ID),
			zap.String("actor_type", ev.Actor.Type),
		)
	}

	if ev.Resource != nil {
		fields = append(fields,
			zap.String("resource_type", ev.Resource.Type),
			zap.String("resource_id", ev.Resource.ID),
		)
	}

	if len(ev.DataSubjects) > 0 {
		fields = append(fields, zap.Strings("data_subjects", ev.DataSubjects))
	}

	if ev.IP != "" {
		fields = append(fields, zap.String("ip", ev.IP))
	}

	if len(ev.Attributes) > 0 {
		fields = append(fields, zap.Any("attributes", ev.Attributes))
	}

	switch severityOf(ev) {
	case SeverityCritical, SeverityHigh:
		log.Error(logMessage, fields...)
	case SeverityWarning:
		log.Warn(logMessage, fields...)
	case SeverityInfo:
		log.Info(logMessage, fields...)
	default:
		log.Info(logMessage, fields...)
	}

	return nil
}

// BrokerSink publishes security events onto the broker event stream (the
// alternative path that fans into the SIEM). Use it where the SIEM ingests from
// the broker rather than the log pipeline.
type BrokerSink struct {
	pub   *broker.Publisher
	topic string
}

// NewBrokerSink returns a BrokerSink that publishes to topic over pub. Pass
// DefaultTopic unless the deployment overrides it.
func NewBrokerSink(pub *broker.Publisher, topic string) *BrokerSink {
	if topic == "" {
		topic = DefaultTopic
	}

	return &BrokerSink{pub: pub, topic: topic}
}

// Emit publishes ev to the configured topic. The event is already stamped and
// validated by the Emitter; broker.Publish is idempotent over the stamp.
func (s *BrokerSink) Emit(ctx *azugo.Context, ev *broker.Envelope) error {
	if s == nil || s.pub == nil {
		return errors.New("secevents: broker sink has no publisher")
	}

	return s.pub.Publish(ctx, s.topic, ev)
}

// EmitBackground publishes ev from a plain context, for a caller with no request.
// It takes the publisher's already-stamped path: the Emitter stamped the event
// before handing it over, and the request-path publish would try to read
// correlation ids off a request that does not exist.
func (s *BrokerSink) EmitBackground(ctx context.Context, ev *broker.Envelope) error {
	if s == nil || s.pub == nil {
		return errors.New("secevents: broker sink has no publisher")
	}

	return s.pub.PublishStamped(ctx, s.topic, ev)
}
