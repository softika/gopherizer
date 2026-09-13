# Security Policy

## Reporting a vulnerability

Please report security issues privately through GitHub's
[private vulnerability reporting](https://github.com/softika/gopherizer/security/advisories/new)
rather than opening a public issue.

Include the affected version or commit, what an attacker can achieve, and the
steps to reproduce it. A minimal reproduction is more useful than a scanner
report.

Expect an acknowledgement within a few working days. Please give us a chance to
ship a fix before disclosing publicly.

## Scope

This repository is a service template. It is intended to be forked and built
on, so the security properties that matter here are the defaults it hands you:

- request handling — input validation, error responses, rate limiting, CORS
- the database layer — query construction, connection handling, migrations
- configuration defaults and the container image
- the build and release workflows

Findings in a downstream service built from this template belong to that
project, unless the cause is a default inherited from here.

## What the template does and does not give you

The template verifies OIDC bearer tokens and enforces per-operation scope and
role requirements, but ships with that **disabled**. Enabling it is
configuration (`oidc.enabled`, `oidc.issuer`, `oidc.audience`) and is the first
thing to do before exposing a service publicly. It does not issue tokens: there
is no login flow, no token endpoint and no user store.

Health probes, `/metrics`, `/docs` and `/openapi.json` are deliberately never
guarded. An orchestrator carries no token, so a probe behind the guard would
take every instance out of rotation during an identity provider outage the
fleet would otherwise survive.

Two properties are enforced in code rather than left to configuration, because
each would otherwise be one environment variable away from disabling
authentication in a deployment:

- The audience, issuer, expiry and signature checks cannot be skipped. go-oidc
  exposes a flag for each; none is reachable from `config`.
- The accepted signing algorithms are asymmetric only, so `alg: none` and HMAC
  key confusion are unrepresentable rather than merely defaulted against.

Tokens carrying `at_hash`, `c_hash` or `nonce` are refused: those are OpenID
Connect ID token claims, and an ID token presented in place of an access token
must not be honoured even when the audience has been misconfigured to an OAuth
client id.

Defaults that are deliberately permissive for local development, and that must
be reviewed before deploying:

| Setting | Default | Why it matters |
| --- | --- | --- |
| `http.cors.origins` | `*` | Narrow to real origins; credentials stay off while this is `*` |
| `http.client_ip.from` | `remote_addr` | Behind a proxy or CDN this buckets every caller together for rate limiting — set `xff` or `header` |
| `http.metrics.enabled` | `true` at `/metrics` | Unauthenticated; restrict it at the ingress or disable it |
| `database.sslmode_disabled` | `true` | Enable TLS for any database that is not on localhost |
| `oidc.enabled` | `false` | Every endpoint is unauthenticated until this is turned on |
| `oidc.require_typed_access_token` | `false` | RFC 9068 `typ` checking is off because not every provider stamps the header; turn it on once yours is confirmed to |
| `database.password` | a placeholder | Supply real credentials via environment or a secret manager, never in the config file |

## Automated checks

Every push and pull request runs `govulncheck` for reachable vulnerabilities in
dependencies, plus `golangci-lint` including `gosec`. `govulncheck` also runs on
a weekly schedule so a new advisory against an unchanged dependency is caught.
Dependency and action updates are proposed by Dependabot.
