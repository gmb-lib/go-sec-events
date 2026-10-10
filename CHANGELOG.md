# Changelog

Notable changes to this library, newest first. Versions are git tags; this file is written
for whoever bumps the dependency — what changed, and what it means for code that already
uses it.

## v1.2.2

A security and dependency release with two things to act on: **this library now needs Go 1.27.2**,
and it **requires azugo v0.40.0 and go-platform-kit v1.11.4**, which your service inherits when it
takes this version. No source changed here and nothing it does behaves differently.

### Changed

- **The module declares `go 1.27.2`** (was `1.27.0`). Go 1.27.2 and `golang.org/x/net` v0.61.0 fix
  vulnerabilities that this library's code reaches under the old set. `govulncheck` found six it
  calls before the move and none after it: GO-2026-6603, -6608, -6611, -6612, -6613 and -6617, in
  `net/http`, `net/textproto`, `mime/multipart` and `golang.org/x/net`. Raise your own module's `go`
  directive to `1.27.2`; from there the go command downloads and uses that toolchain by itself. CI
  that reads `go-version-file: go.mod` follows with no workflow edit.

- **`azugo.io/azugo` and `azugo.io/core` → v0.40.0** (were v0.38.1), with **`go-platform-kit` →
  v1.11.4**. Nothing in this library's API changed with them. What azugo v0.40.0 changes for a
  service (outbound calls stop at the request deadline, cookies found by their bare name, `303`
  after a non-GET redirect, a wider cache interface, one more cache at start) is listed in the
  `go-platform-kit` v1.11.4 notes.

### Notes

- **Also moved:** `fasthttp` → **v1.75.0**, `golang.org/x/net` → **v0.61.0**, and the indirect
  modules that came with the platform kit, OpenTelemetry → v1.47.0 and gRPC → v1.84.0 among them.

- **CI's linter moved to golangci-lint v2.14.0** (was v2.13.1). The earlier release cannot read Go
  1.27.2's compiled standard library and stops before linting anything. CI's pinned GitHub Actions
  also moved to their current commits. No code changed with either.

- The gate is green on Go 1.27.2: `go mod verify`, `go mod tidy -diff`, build, vet, `gofmt`,
  golangci-lint v2.14.0, `go test -race` with **0 races**; `govulncheck` reports nothing this library
  calls.

## v1.2.1

Dependency maintenance with one thing to act on: **this library now needs Go 1.27**. No source
changed here — but one piece of documentation was wrong about a regulatory deadline, and the
platform kit moves two patch releases in one step with an upstream behaviour change riding along.
See *Fixed* and *Notes*.

### Changed

- **The module declares `go 1.27.0`** (was `1.26.6`), so your own module has to be on Go 1.27
  before it can build against this one. A dependency's `go` line does **not** make the go command
  fetch a newer toolchain for you — measured both ways: a consumer whose own `go` directive is
  lower stops with a `requires go >= 1.27.0 (running go 1.26.6)` error, and it stops there with
  `GOTOOLCHAIN` on its `auto` default just as it does under `local`. Raise your own `go` directive
  to `1.27.0` first; from there the go command downloads and uses the 1.27 toolchain by itself, so
  nobody has to install Go by hand. CI that reads `go-version-file: go.mod` follows the bump with
  no workflow edit — a workflow naming a Go version in the YAML needs that line changed.

### Fixed

- **The documented reporting clock said the final report is due 30 days after awareness. It is due
  one month after the *notification*.** The shorthand "24-72-30" the README, `SECURITY.md` and the
  package documentation all used compressed three deadlines with two different anchors into one
  string, and got the last one wrong: an early warning within 24 h and a notification within 72 h
  of first awareness, then a final report **within one month of that notification**. No code
  changed — `FirstAwareness` captures and returns the awareness instant exactly as before — but if
  you built an incident-register deadline off the old wording, it was computed from the wrong
  anchor. The shorthand is now spelled out as `24 h / 72 h / 1 month` wherever it appeared.

### Notes

- **`github.com/gmb-lib/go-platform-kit` → v1.11.3** (was v1.11.1), and with it the framework:
  `azugo.io/azugo`, `azugo.io/core` and `azugo.io/opentelemetry` → **v0.38.1**,
  `github.com/valyala/fasthttp` → **v1.74.0**. Nothing this library takes from any of them moved —
  it has no metrics code and touches fasthttp only in a test.

  **One thing in that framework release is visible to your monitoring, not to your code, and it
  arrives with nothing to opt into.** From azugo v0.38.1 the metrics endpoint no longer negotiates
  OpenMetrics: a scraper sending `Accept: application/openmetrics-text` is answered
  `Content-Type: text/plain; version=0.0.4; charset=utf-8` with no `# EOF` terminator, where it used
  to get the OpenMetrics format. **The metric names, labels and values are unchanged.** It reaches a
  service through the platform kit, which binds azugo's metrics configuration — so if your scrape
  configuration demands the OpenMetrics content type, or treats a missing `# EOF` as a truncated
  scrape, **check it before you deploy**.

- **One module leaves the dependency graph and another joins it**, both transitively through
  fasthttp: `github.com/andybalholm/brotli` is gone and `github.com/molecule-man/go-brrr` v1.1.0
  provides the brotli implementation in its place. Also moved indirectly:
  `go-playground/validator/v10` → v10.30.4, `klauspost/compress` → v1.20.0, `golang.org/x/crypto` →
  v0.57.0, `x/net` → v0.59.0, `x/sys` → v0.48.0, `x/text` → v0.42.0, and the
  `google.golang.org/genproto/googleapis/{api,rpc}` snapshots → 20260911204522. Nothing in this
  library calls any of them directly.

- The repository gained a code of conduct, and the advisory DCO workflow was removed now that the
  sign-off is enforced by the organisation's app together with a branch ruleset. What a contribution
  has to carry is unchanged.

- The gate is green on the new set: `go mod verify`, `go mod tidy -diff`, build, vet, `gofmt`, and
  `go test -race` with **0 races**; `govulncheck` finds nothing.

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
