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

The template ships with no authentication or authorization. Adding them is the
responsibility of whoever builds on it, and is the first thing to do before
exposing a service publicly.

Defaults that are deliberately permissive for local development, and that must
be reviewed before deploying:

| Setting | Default | Why it matters |
| --- | --- | --- |
| `http.cors.origins` | `*` | Narrow to real origins; credentials stay off while this is `*` |
| `http.client_ip.from` | `remote_addr` | Behind a proxy or CDN this buckets every caller together for rate limiting — set `xff` or `header` |
| `http.metrics.enabled` | `true` at `/metrics` | Unauthenticated; restrict it at the ingress or disable it |
| `database.sslmode_disabled` | `true` | Enable TLS for any database that is not on localhost |
| `database.password` | a placeholder | Supply real credentials via environment or a secret manager, never in the config file |

## Automated checks

Every push and pull request runs `govulncheck` for reachable vulnerabilities in
dependencies, plus `golangci-lint` including `gosec`. `govulncheck` also runs on
a weekly schedule so a new advisory against an unchanged dependency is caught.
Dependency and action updates are proposed by Dependabot.
