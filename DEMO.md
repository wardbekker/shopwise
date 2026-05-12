# Webinar Demo: Alert → Assistant Investigation → Skill → Workspace → Incident → PR

End-to-end AI-driven incident response loop in Grafana Cloud, driven by the `webinar_demo` Go microservices app. A fault surfaces as an alert, IRM auto-declares an incident, IRM auto-triggers a Grafana Assistant Investigation attached to that incident, the Investigation matches a preprovisioned Skill via semantic search, the engineer reviews the workspace, and the Assistant opens a PR to fix the root cause — all without leaving Grafana.

## Demo Flow → Grafana Features

| Step | Demo beat | Grafana product feature |
|------|-----------|-------------------------|
| 1 | Alert fires | Grafana managed alert rule + contact point routed to IRM |
| 2 | Incident auto-declared | IRM **auto-declare rule** on the OnCall alert group → IRM incident |
| 3 | Investigation auto-triggers, attached to the incident | IRM outgoing webhook: **"Grafana Assistant for IRM"** preset, trigger **Incident Created** |
| 4 | Investigation uses preprovisioned Skill (embedding match) | Grafana Assistant **Skills** (GA Mar 2026), "Visible to agents" toggle → semantic search index |
| 5 | Work in workspace | Assistant **Investigation workspace** (timeline, agent findings, report) — opens from the incident page |
| 6 | Assistant opens PR | **Aspirational** — framed as Assistant directly creating a GitHub PR (not a documented Grafana capability today; narrate as forward-looking) |

## Assets to Provision

### A. Application fault (in this repo, ahead of demo)

Inject a fault in a Go service on a branch — e.g., bump error rate in `services/payment` or `services/checkout`. Run `loadgen` to surface it. The fault and its fix are the substance of the PR in step 6.

- Branch: `demo/payment-bug` (or similar)
- Fix branch (target of step 6 PR): `main`
- Fault should be obvious enough that a fix is a small, demo-friendly diff (e.g., an off-by-one, a missing nil check, a wrong threshold)

### B. Grafana Cloud assets

| Asset | Provision via | Notes |
|-------|---------------|-------|
| Prometheus alert rule (e.g., payment error rate > X) | `gcx alert rules` (push manifest) | Verify with `gcx alert rules list` |
| Contact point → IRM | `gcx alert contact-points` | IRM preset |
| IRM integration receiving the alert | `gcx irm oncall integrations` | Grafana Alertmanager integration type |
| Escalation chain / route | `gcx irm oncall` | Minimal — no human paging needed for the demo |
| **IRM auto-declare rule** (alert group → incident) | **Grafana UI** | IRM → Incidents → Settings → Auto-declare → match on alert-group labels (`service=payment`, `severity=critical`) and pick incident severity. **Not surfaced via `gcx irm incidents` today** (only `create`, `list`, `get`, `close`, `activity`, `severities` exist). |
| **IRM outgoing webhook — "Grafana Assistant for IRM" preset** | **Grafana UI** | Alerts & IRM → IRM → Integrations → Outgoing webhooks → New webhook → select preset. Use trigger **"Incident Created"** so the Investigation is attached to the incident (not the alert group). Not currently surfaced via `gcx`. |
| **Assistant Skill** (preprovisioned, "Visible to agents" on) | **Grafana UI** | Grafana Assistant → Skills → New Skill. Title + description + structured content (headings = procedure steps) + `@`-attached dashboards/queries. Visibility: Team. Not currently surfaced via `gcx`. |

### C. Skill content (preprovisioned)

Write one Skill that the investigation will semantically match for this fault. Example — title: *"Investigate payment service error spikes"*. Content (markdown with headings) walks through:

1. Confirm the alert/symptom (link a dashboard via `@`)
2. Check error-rate metric (link the PromQL query)
3. Pull recent logs (link the LogQL query)
4. Correlate with recent deploys
5. Recommend remediation

The Skill is what makes step 3 land in the demo: when the Investigation kicks off, the semantic search index returns this Skill because the alert summary matches its title/description embedding, and the Assistant follows the procedure.

## Demo Script (narration)

1. **Set the stage.** Show the `webinar_demo` topology dashboard. Mention the Skill that's preprovisioned — open the Skills list briefly.
2. **Trigger the fault.** Deploy the `demo/payment-bug` branch (or flip its env var), start loadgen.
3. **Alert fires.** Show it in Alerting → Active. Show that the contact point routed it to IRM.
4. **Incident auto-declares.** Switch to IRM → Incidents; the new incident appears without anyone clicking. Point out the auto-declare rule.
5. **Investigation auto-triggers, attached to the incident.** On the incident page, the Grafana Assistant Investigation appears in the activity / sidebar. Open the workspace. Point out that *no one clicked anything* — the webhook fired on Incident Created.
6. **Skill matched.** In the workspace timeline, highlight where the Assistant references the preprovisioned Skill (semantic match — no `@`-mention needed). Walk through the agent fan-out: metrics, logs, traces, profiles.
7. **Review the report.** Key findings, timeline, recommended next steps.
8. **PR via coding agent.** The Investigation's RCA stops at *"bad deploy → rollback"* — it correctly recommends the safest immediate action but doesn't reach the source. Switch to Claude Code in the repo: it reads `gcx assistant investigations narrative <id>` (the Investigation's executive summary) plus `services/payment/main.go`, identifies that `pickProcessor` returns a nil `*paymentProcessor` for `amount > 100` when `BUG_AMOUNT_PANIC=1`, and opens a PR that either finishes wiring the high-value processor or removes the bug toggle path. Pre-prepare the diff so the reveal lands fast. Honest framing: Grafana Assistant doesn't open the PR itself today — a coding agent grounded on the Investigation's output does.

## Provisioning Order (pre-demo checklist)

1. Verify gcx context: `gcx config check`
2. Create Prometheus alert rule (gcx)
3. Create IRM integration + route (gcx)
4. Create contact point → IRM (gcx)
5. **UI:** Create IRM auto-declare rule on the integration (match alert-group labels → declare incident with chosen severity)
6. **UI:** Create IRM outgoing webhook ("Grafana Assistant for IRM" preset) with trigger **Incident Created**
7. **UI:** Create the Assistant Skill with "Visible to agents" enabled, attach the relevant dashboard and queries
8. Build & deploy the `demo/payment-bug` branch to the cluster
9. Dry-run: trigger fault → alert fires → incident auto-declared → Investigation appears on the incident → Skill referenced in workspace
10. Prepare the "fix" PR on GitHub (draft) for the step-8 reveal

## Verification

End-to-end check before the webinar:

- `gcx alert rules list` — rule present and healthy
- `gcx alert instances list --state firing` — fires under load
- `gcx irm oncall integrations list` — integration receives the alert group
- `gcx irm incidents list` — new incident appears (auto-declared) within ~30s of the alert group
- Grafana UI → Outgoing webhooks — "Grafana Assistant for IRM" webhook shows a recent **Incident Created** delivery
- Grafana UI → IRM → Incidents → open the new incident — the Investigation is visible in the incident's activity / sidebar without manual action

Read the Investigation from the CLI (lodestone variant — the legacy `report`/`timeline`/`todos` subcommands return empty stubs and should not be used):

```sh
INV=$(gcx irm incidents list --json metadata.name,spec.refs -o json \
  | jq -r '.[0].spec.refs[] | select(.key=="com.grafana.assistant") | .ref')

gcx assistant investigations narrative $INV   # agent prose / executive summary
gcx assistant investigations skills    $INV   # confirms the Skill semantic match
gcx assistant investigations tools     $INV   # full tool-call trail (Prom/Loki/Tempo)
gcx assistant investigations chat      $INV   # raw message thread, if needed
```

Requires the gcx build that contains the lodestone subcommands (commit `85e6036d` or later). Earlier builds expose only the v1 surface, which returns empty content on lodestone investigations.

## Open Items / Risks

- **Skill provisioning is UI-only today.** If multiple Skills need to be reproduced, capture screenshots / export the content as markdown for re-creation. Watch for a future `gcx` surface.
- **Auto-declare rule is UI-only today.** Same caveat — screenshot the rule and document the match labels + severity so it can be rebuilt by hand if the stack is rebuilt.
- **Step 8 PR creation is coding-agent-driven, not Assistant-driven.** Grafana Assistant doesn't open PRs today. Step 8 in this script uses Claude Code, grounded on `gcx assistant investigations narrative <id>` plus the source tree. Pre-prepare the diff for the reveal.
- **Semantic match is non-deterministic.** Rehearse the alert summary / Skill title wording so the match is reliable. If it doesn't match in a rehearsal, tighten the Skill's title/description until it does.
- **Stale-data anchoring across demo runs.** Loki retains panic logs from earlier toggles; an Investigation run later in the day will identify the *earliest* panic of the day, not the one that just fired, and cite stale pod names. Mitigation: run the webinar as the first fault of the day, or `make down && make up` an hour before to flush log history.

## References

- [Skills in Grafana Assistant are now GA](https://grafana.com/whats-new/2026-03-17-skills-in-grafana-assistant-are-now-generally-available/)
- [Create Skills — Grafana Cloud docs](https://grafana.com/docs/grafana-cloud/machine-learning/assistant/guides/skills/)
- [Configure IRM webhooks for automated investigations](https://grafana.com/docs/grafana-cloud/machine-learning/assistant/configure/irm-webhooks/)
- [Run investigations — Grafana Cloud docs](https://grafana.com/docs/grafana-cloud/machine-learning/assistant/guides/investigation/)
