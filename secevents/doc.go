// Package secevents is the NIS2-audit (NIS2 security-operations) event emitter for
// eIDAS signing services. It gives every service one standard way to emit
// structured security events — auth failures, authZ/IDOR denials, DPoP/proof
// failures, egress/NetworkPolicy violations, secret/key access, privileged/admin
// actions, and "first-awareness" incident detections — to the SIEM / central log
// management.
//
// Events are the frozen broker.Envelope tagged broker.CategorySecurity,
// stamped with a ULID id, a high-precision occurrence time, and the request's
// correlation/trace ids. A pluggable Sink decides where they go: a LogSink emits
// them as structured log lines the platform's log pipeline ships to the SIEM (the
// common path), or a BrokerSink publishes them onto the event stream. The library
// is decoupled from the concrete transport so it stays in-process glue.
//
// # NIS2 timing (24 h / 72 h / one month)
//
// The occurrence time is a high-precision, synced-clock instant so the NIS2
// reporting clock is defensible: an early warning within 24 h and a notification
// within 72 h of "when we first became aware", then a final report within one
// month of that notification — the last deadline runs from the notification, not
// from awareness. FirstAwareness captures the awareness instant explicitly and
// returns it, so the caller can record it in the incident register.
//
// # PII posture
//
// Security events are mostly metadata; pseudonymise actors where possible and keep
// the envelope content-free. The emitter strips free-text/content attribute keys
// defensively, and the publisher strips bearer-token-shaped keys.
//
// eIDAS-audit (signing evidence) and GDPR-audit (GDPR access) are separate mechanisms
// with their own libraries (go-eidas-audit, go-gdpr-audit). A single action may be
// security-relevant *and* a GDPR access (e.g. operator break-glass): the service
// emits both, rather than overloading one event.
package secevents
