# Skill: Investigate payment service error spikes

Paste the fields below into **Grafana Assistant → Skills → New Skill**. Set **Visible to agents** ON and **Visibility** to Team so Investigations can semantic-match it.

This Skill expresses **intent and process** — what to check and what good looks like — not concrete queries or dashboards. The Assistant's specialist agents will discover the right signals from your stack.

---

## Title

```
Investigate payment service error spikes
```

## Description

Runbook for elevated errors or latency on the `payment` service in the webinar-demo e-commerce stack. Use when payment alerts fire, checkout success rate drops, or order throughput stalls.

## Slash command (optional)

```
/payment-errors
```

## Content (paste into the Skill body)

```markdown
# Investigate payment service error spikes

You are helping a responder triage elevated errors on the `payment` service
in the webinar-demo stack. Work through the steps below in order and surface
findings as you go. At each step, state what you looked at and what you found
before moving on.

## 1. Confirm the symptom

Establish that there really is a problem on `payment`:

- Is the server-error rate elevated versus a normal baseline?
- Or is request latency above its usual range?

If neither is true, stop and report a likely false positive with the evidence
that ruled it out.

## 2. Quantify the blast radius

Characterise the impact, not just the symptom:

- How bad is the deviation (multiple of baseline, error percentage)?
- When did it start, and is it still ongoing or already recovering?
- Is it isolated to `payment`, or are upstream callers (`checkout`,
  `order`, `frontend`) seeing knock-on errors?
- Which routes or operations are affected — all of them, or just one?

## 3. Inspect logs and traces

Look for a cause signature, not just more symptoms:

- Is there a dominant error message, exception, or status code?
- Are there panics, nil dereferences, or context-deadline-exceeded patterns?
- Do failing requests share a trace pattern (a slow downstream call, a
  retry storm, a saturated dependency)?
- Can you tie failures to specific request shapes (a customer, a region,
  a payload variant)?

## 4. Correlate with recent changes

Most production incidents are caused by a recent change. Check:

- Recent deploys of `payment` and its direct dependencies
- Recent config, feature-flag, or secret rotations
- Recent infra changes (node pool, network policy, autoscaler)

Align change timestamps with the incident start time (±2 minutes). Name the
suspect change if one aligns; explicitly say "no aligned change found" if not.

## 5. Recommend remediation

Pick the lightest action that addresses the root cause:

- **Rollback** if a recent change clearly aligns with the spike
- **Hotfix** if the root cause is obvious from logs/traces and the diff is
  small and well-scoped — describe the change in one or two sentences
- **Scale or capacity** if the cause is load, not a code or config change
- **Escalate** if evidence is inconclusive — say what's missing

## Reporting standard

End the investigation with:

- **Suspected root cause** — one sentence in the summary, naming the
  specific code location, config, or change when the evidence supports it
- **Recommended actions** — ordered by impact, with the first being the
  immediate mitigation (rollback, hotfix, scale, or escalate) and the rest
  scoped to follow-up. Say "no aligned change found" or "evidence
  inconclusive" rather than padding the list with speculation.
- **Evidence** — what you looked at (queries run, panels consulted, traces
  inspected, log signatures quoted), so a human can re-verify quickly
- **Ruled-out hypotheses** — a short table of what you considered and
  disproved, with the disproving evidence
```

## Why this Skill matches semantically

The Investigation receives the alert summary as its prompt. Keywords in the title, description, and headings — *payment*, *error*, *latency*, *checkout*, *order* — drive the embedding match. If a rehearsal misses, tighten the wording to echo the alert summary, not the query syntax.
