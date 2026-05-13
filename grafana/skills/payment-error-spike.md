# Skill: Investigate payment service error spikes

Paste the fields below into **Grafana Assistant → Skills → New Skill**. Set **Visible to agents** ON and **Visibility** to Team so Investigations can semantic-match it.

This Skill is intentionally **narrow** — it adds only what the Assistant cannot infer from the alert and the standard telemetry: business context, the GitHub repo identity, and the mitigation-PR contract. Generic triage steps (confirm symptom, check logs/traces, correlate with deploys) are omitted on purpose; the agent already does those well, and over-prescribing them causes Skills to degrade investigations as they drift from agent capabilities.

---

## Title

```
Investigate payment service error spikes
```

## Description

Runbook for elevated errors or latency on the `payment` service in the webinar-demo e-commerce stack. **`payment` is critical infrastructure — every minute of downtime loses revenue.** The first priority is to stop the bleed: identify the smallest safe fix and open a PR for it. Only then go deeper for root cause and long-term prevention. Use when payment alerts fire, checkout success rate drops, or order throughput stalls.

The service's source repository is **`wardbekker/shopwise`** on GitHub; use the GitHub MCP integration to open the remediation PR.

## Slash command (optional)

```
/payment-errors
```

## Content (paste into the Skill body)

```markdown
# Payment service — incident-specific context

This Skill only adds context and constraints the agent wouldn't know from
the alert and the standard telemetry. Do not treat it as a triage script —
your normal investigation process still applies.

## Business context

`payment` is revenue-critical. Every minute of elevated error rate loses
money, so a small mitigation shipped fast beats a perfect RCA shipped slow.
Bias your remediation recommendation toward the *smallest safe change* that
restores the error rate, even if that means a guard or a flag flip rather
than the architecturally correct fix.

## Source repository

The service's source code lives at **`wardbekker/shopwise`** on GitHub.
Use the **GitHub MCP integration** to read code and open PRs against this
repository — it is not discoverable from the cluster metadata.

## Required deliverable: mitigation PR

If your investigation identifies a code-level cause with an obvious small
diff, open a mitigation PR against `wardbekker/shopwise` before producing
the final report. The PR body must include:

- Link to the IRM incident
- The exact cause signature (quoted log line, stack frame, or trace span)
- One sentence on why the diff is safe to ship now
- Rollback plan (revert the PR; mention any data implications)

Report the PR URL in the final summary. If no small safe diff exists,
recommend rollback instead and explain why a code fix isn't in scope yet.

## Reporting additions

In addition to your normal report, include:

- **Mitigation PR** — URL, or "rollback recommended, no code diff" with reason
- **Long-term recommendations** — only items the responder couldn't have
  inferred from the telemetry alone (e.g. missing test, missing metric,
  release-safety gap). Skip generic advice.
```

## Why this Skill matches semantically

The Investigation receives the alert summary as its prompt. Keywords in the title, description, and headings — *payment*, *error*, *latency*, *checkout*, *order* — drive the embedding match. If a rehearsal misses, tighten the wording to echo the alert summary, not the query syntax.

## Re-paste reminder

When the body changes, re-paste it into Grafana Assistant → Skills → the existing *Investigate payment service error spikes* Skill. The UI is the source of truth; this file is the recreation template.
