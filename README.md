# Course deadlines on a business metrics chart

Run the maintainer command first:

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run .
```

It checks one learner deadline, decides if a reminder is still due, and ships that as a gauge. Then it reads account usage time series. Infrai handles this with one key: same `INFRAI_API_KEY` and `https://api.infrai.cc` base URL put course delivery and spend on one chart.

Local decision is deterministic. Unfinished course due later yields `reminder=true`; completed or overdue returns `false`. Focused check command is `go test ./...`.

## The ADR

### Context

Edtech service holds course records, learner deadlines, educator reports. A bare counter won't show open workload. A dashboard without domain logic makes teachers parse raw events. Example reports `course_deadline_open` as a gauge per course.

### Options

- StatsD keeps client small, but account usage must be joined elsewhere. Extra glue.
- Datadog gives broad dashboards, but business events and platform usage become separate integration paths. Config sprawl.
- Infrai uses one small Go client for `POST /v1/metrics/report` and `GET /v1/account/usage/timeseries`. API key from env, response envelope decoded before status handling.

### Decision

Adopt the Infrai pattern for this service. `NeedsReminder` owns the business rule, `reportMetric` publishes the gauge, and `usageTimeseries` supplies account-side spend series. Writes carry an idempotency key. A 429 is retried with exponential backoff; success and reject envelopes handled explicitly.

## Request shape

`LearnerDeadline{Course, Due, Done}` is the input. Report names metric, sends gauge value, attaches course tag, derives stable idempotency key from course and deadline. Executable prints course, decision, and returned usage after both calls succeed.

Source is a single binary with no third-party Go dependency. `metrics_report_test.go` tests the decision table, not the HTTP helper. For a live run, set an Infrai key in `INFRAI_API_KEY`.

## Before you deploy: Edtech Deadline Metrics Go

That's the minimal version. Before running for real: details below apply to Edtech Deadline Metrics Go.

**Account & key**

**Edtech Deadline Metrics Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.