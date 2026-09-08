# Update-failure monitoring

Enable the HTTP Prometheus service with `--service-prometheus-enabled` (disabled
by default), then configure your monitoring system to scrape each instance.

| Metric | Meaning |
| --- | --- |
| `contentserver_last_successful_update_timestamp_seconds` | Completion time of the latest successful update or unchanged-content check |
| `contentserver_last_failed_update_timestamp_seconds` | Completion time of the latest failed update attempt |

Both gauges export Unix seconds with fractional precision and start at `0`.
They belong to the process-wide metrics package and assume one repository per
process. They add no application labels or persisted state. Prometheus attaches
the target labels when scraping.

A successful load, HTTP 304, or unchanged-version response advances success.
Fetch, parsing, and dimension-load errors advance failure. Restoring cached
content does not establish upstream success. Busy-request rejections are not
update attempts. History persistence errors use the separate
`contentserver_history_persist_failed_count` counter and do not turn an otherwise
successful update into a failure. Existing update counters remain available.

## Alert rule

Load [alerts.yml](alerts.yml) through the owning Prometheus configuration's
`rule_files` setting. This repository supplies an example; it does not install
the rule or configure scraping or notification routing.

The rule compares the last failure with the last success on each target and uses
`for: 5m` to alert when a failure remains unrecovered for five minutes of rule
evaluations. An initial failure can alert before any successful load. Repeated
failures keep the condition active without restarting the delay. A successful
retry clears the condition at the next evaluation. Scrape and evaluation
intervals affect when changes become visible.

Preserve each target's labels. Aggregating success across replicas could mask a
failing instance. The gauges detect completed update failures; stalled polling,
missing targets, and failures to generate fresh upstream content need separate
signals.

## Restarts

An application restart resets both gauges to `0`; snapshot restoration does not
restore them. Prometheus clears pending or firing state when an evaluation
observes a false condition or an absent series. If the process fails again before
Prometheus evaluates the reset or absence, the condition can remain continuously
active and retain its previous pending time. A restart alone does not guarantee
a fresh five-minute delay. This describes application restarts, not restarts of
the Prometheus server itself.

## Validation

From the repository root, with `promtool` installed:

```sh
promtool check rules docs/monitoring/alerts.yml
promtool test rules docs/monitoring/alerts.test.yml
```

The fixtures cover startup zeros, delayed firing, repeated failures, recovery,
independent replicas, and evaluated versus unobserved restart resets.

The metric model follows Prometheus guidance on
[timestamps](https://prometheus.io/docs/practices/instrumentation/#timestamps-not-time-since),
and the delay uses its
[alerting rule semantics](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/).

## Rollback / Reverse Plan

If validation fails, revert the instrumentation and example-rule change before
release. No content migration or data loss is involved. If the example is later
installed, revert the monitoring configuration change separately; rule removal
takes effect after configuration reload and evaluation.
