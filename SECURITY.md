# Security policy

This library is how every service in a fleet emits its security events — authentication failures,
authorization denials, proof failures, egress violations, secret and key access, privileged
actions, and the **first-awareness** detection that anchors the NIS2 24-72-30 reporting clock. It
is the code an investigation reads afterwards, and the code a regulator's timeline is built from.
Both make its failure modes distinctive: an event that is missing or wrongly timed is a reporting
failure, not a logging inconvenience — and an event that carries the credential it is reporting on
turns the security log into the thing that needs reporting.

Please report security problems privately. Do not open a public issue, pull request or discussion
for anything that could be exploited before a fix exists.

## How to report

Use **[private vulnerability reporting](https://github.com/gmb-lib/go-sec-events/security/advisories/new)**
on this repository. The report stays visible only to you and the maintainers until an advisory is
published, and it gives us one place to discuss and co-ordinate a fix with you.

Please include, as far as you can establish it:

- what the problem is, and what an attacker or an unintended reader gains from it;
- the smallest set of steps that reproduces it, and against which version or commit;
- which sink was in use, and the configuration it needs if it only appears under particular
  settings;
- whether you have told anyone else, and whether a disclosure date already binds you.

Please redact anything real — a placeholder event explains a leak finding better than a live one.

## What happens next

- We acknowledge a report within **five working days**.
- We tell you whether we can reproduce it, and what we think its severity is, as soon as we know.
- We keep you updated while a fix is prepared, and we agree a disclosure date with you. Our default
  is to publish an advisory once a fix is available, and in any case within **90 days** of the
  report — earlier if the problem is already public or being exploited.
- We credit you in the advisory unless you would rather stay anonymous.

There is no bug-bounty programme. We are grateful anyway, and we say so publicly.

## What we consider most serious

- An event that is not emitted at all, or a sink that drops events while the caller is told the
  emit succeeded — the security log then says nothing happened, which is indistinguishable from
  nothing having happened.
- A wrong or imprecise occurrence time on a first-awareness detection. The NIS2 clock is anchored
  on that moment, so an incorrect anchor is a reporting defect with a legal consequence, and clock
  handling here deserves more suspicion than it would anywhere else.
- The event carrying what it reports on: a credential, token, proof, key or secret in an attribute,
  or a direct identifier where `DataSubjects` requires a pseudonymous internal reference.
- An event attributed to the wrong actor, service, tenant or request, so an investigation reaches
  the wrong conclusion about who did what.
- A way for a caller — or an untrusted value a caller passes through — to forge, suppress or replay
  events, including log-injection through a field that reaches a log line unescaped and lets a
  forged record be read as genuine by the pipeline.
- A log-sink line an operator's pipeline cannot reliably tell apart from ordinary application
  logging, so security events are lost in the noise rather than routed to the SIEM.

Denial of service and findings that need an already-compromised host are in scope but lower
priority. Reports about outdated dependencies are welcome where you can show the vulnerable path
is actually reachable.

## Scope

This policy covers the code in this repository. It does not cover the log pipeline, broker or SIEM
a deployment ships events to, the time source the host clock is synchronised from, or the services
that import this library — report those to the parties that operate them.

## Supported versions

Security fixes land on the most recent release. Older tags are not patched; if you are pinned to
one, the fix is to move forward. This module is pinned in lockstep with the platform kit, so a fix
may require moving that pin too.
