# Grafana Cloud demo manifests

Source-of-truth definitions for the Grafana Cloud assets that back the demo
flow (`docs: ../../DEMO.md`). Sanitised — no tokens, no server-generated IDs
where avoidable.

## Layout

```
manifests/
  folder.yaml                          # Grafana folder hosting the alert rule
  alerting/
    alert-rule.yaml                    # PromQL alert on payment 5xx ratio
    contact-point.yaml                 # OnCall contact point (URL templated)
    notification-policy.yaml           # Root policy with the payment route
  irm/
    integration.yaml                   # OnCall integration (type grafana_alerting)
    escalation-chain.yaml              # Minimal chain (no human steps)
    webhook-grafana-assistant.yaml     # Outgoing webhook (grafana_assistant preset)
```

The Grafana Assistant **Skill** is at `../skills/payment-error-spike.md` — that
asset is UI-only and must be pasted into Grafana Assistant → Skills.

## Apply order

OnCall resources are server-generated and IDs feed downstream — order matters:

1. `folder.yaml` → note the returned `uid`, set it on `alert-rule.yaml.folderUID`
2. `irm/integration.yaml` → note `id` and `integration_url`
3. `irm/escalation-chain.yaml` → note `id`
4. Bind chain to the integration's default route (see header in
   `irm/escalation-chain.yaml`)
5. `irm/webhook-grafana-assistant.yaml` → substitute integration id in
   `integration_filter`
6. `alerting/contact-point.yaml` → substitute the integration URL
7. `alerting/alert-rule.yaml`
8. `alerting/notification-policy.yaml`

Each manifest has the exact `gcx`/`gcx api` invocation in its header.

## Current provisioned IDs

Live IDs in stack `wbkprez` are listed in `../PROVISIONED.md`. Use those
when patching or deleting; use the manifests here when recreating from
scratch.

## Why some manifests have placeholders

- `contact-point.yaml` → `__INTEGRATION_URL__` (contains an auth token; never
  committed verbatim per `../../CLAUDE.md`)
- `webhook-grafana-assistant.yaml` → `__INTEGRATION_ID__` (server-assigned;
  resolved from the integration's `metadata.name`)
- `alert-rule.yaml` → `folderUID` references a real folder; update after
  applying `folder.yaml`
