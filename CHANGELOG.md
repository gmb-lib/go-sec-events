# Changelog

Notable changes to this library, newest first. Versions are git tags; this file is written
for whoever bumps the dependency — what changed, and what it means for code that already
uses it.

## v1.2.0 — a security event can come from work with no request behind it

Additive: `Sink`, `Emit`, `NewLogSink` and `NewBrokerSink` all keep their signatures, so existing
code compiles and behaves unchanged.

**Requires `go-platform-kit` v1.11.0 or later** — the background stamp depends on
`broker.Stamp` tolerating a nil request, which that release fixes.

### Added

- **`Emitter.EmitBackground(ctx context.Context, ev *broker.Envelope) error`** — the entry point
  for a scheduled sweep, a purge, a drainer. It tags, sanitizes, stamps and validates exactly as
  `Emit` does. The one difference is that the event carries no correlation or trace id, because
  there is no request to take them from.

- **`BackgroundSink`**, the optional second half of `Sink`:

  ```go
  type BackgroundSink interface {
      EmitBackground(ctx context.Context, ev *broker.Envelope) error
  }
  ```

  A separate interface, not a second method on `Sink`, so that no existing implementation breaks —
  including the `capturingSink` test doubles services keep. A sink without it is reported plainly
  (`sink cannot deliver background events`) rather than silently dropping the event.

- **`NewLogSinkFor(log *zap.Logger)`** beside `NewLogSink()`. The request path borrows the
  request's logger; background work has none to borrow, so a sink that must serve it is built with
  one of its own. Pass the service's own logger.

  ```go
  // serves both paths
  audit := secevents.NewEmitter(secevents.NewLogSinkFor(a.Log()))
  ```

  `NewLogSink()` is unchanged and still correct for a service that only emits on the request path;
  calling `EmitBackground` on one says so instead of panicking.

- **`BrokerSink` really publishes background events**, through the publisher's already-stamped
  path. This closes a hole rather than adding a convenience: before, a deployment on the broker
  sink got request-path events on the broker while background events had nowhere to go at all — so
  a record written by a scheduled job, which is exactly the kind that proves a retention policy
  ran, would not arrive where it was configured.

### Changed

- **The log line now carries `operation`** when the event sets one. Without it a deletion was
  indistinguishable from any other event on the log path, and a SIEM rule had to infer the act
  from the event type alone. Purely additive to the line; existing queries are unaffected.

- The request and background paths now share **one** field builder inside the library, so the two
  cannot drift apart.

### Why it exists

Every part of the request path reaches into the `*azugo.Context`: the log sink borrows the
request's logger, the broker sink's publish reads the correlation ids bound to the request, and
the stamp reads them too. A caller without a request had no safe way in — passing nil was a
crash, not a degradation — so services wrote the sink's log line themselves and copied its field
names. Five had done so. A SIEM selects on exactly those names, and copies drift.

## v1.1.4

### Changed

- **`azugo.io/azugo` and `azugo.io/core` → v0.38.0, `github.com/gmb-lib/go-platform-kit` →
  v1.10.0.** No source change here: the platform-kit release is additive — a size cap on a
  JetStream stream, which this library does not configure — and nothing else reaches this code.

  One thing in the framework release is worth knowing if you use azugo directly: `user.Basic`'s
  `MarshalJSON` **moved to a pointer receiver**, so marshalling a `Basic` *value* silently produces
  default field JSON instead of the custom form — no compile error.

### Notes

- The repository gained the open-source kit it was missing — `SECURITY.md`, `CONTRIBUTING.md`,
  a secret-scan configuration and the README sections pointing at them — plus this file.

---

The entries below were **reconstructed from git history** rather than written at the time, so they
say what each tag contains, not why it was decided.

## v1.1.3 · v1.1.2 · v1.1.1

- Dependency updates only.

## v1.1.0

- No library change: continuous-integration and linter configuration, a dependency-review workflow,
  and dependency updates. The API is identical to v1.0.2.

## v1.0.2 and earlier

- Not reconstructed. See the git history and the tag list.
