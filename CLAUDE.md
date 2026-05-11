# Project rules

- Never commit sensitive data. This includes secrets, API tokens, OAuth tokens, passwords, private keys, customer data, and Grafana Cloud stack tokens. Treat `.env`, anything under `grafana/PROVISIONED.md`-style notes that include URLs with embedded tokens, and any file matching `*token*`, `*secret*`, `*credentials*`, `*.pem`, `*.key` as suspect — read before staging. If in doubt, ask before `git add`.
