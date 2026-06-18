<h1 align="center">statra</h1>

<p align="center">
  <b>One CLI for App Store Connect + Google Play Console stats.</b><br>
  Pull downloads, revenue, crashes, ANR &amp; ratings for <b>iOS and Android</b> in a single command.
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/built%20with-Go-00ADD8?logo=go&logoColor=white">
  <img alt="License: MIT" src="https://img.shields.io/badge/license-MIT-green">
  <img alt="Platforms" src="https://img.shields.io/badge/stores-App%20Store%20%2B%20Google%20Play-black">
</p>

---

## What is statra?

**statra** is an open-source command-line tool that pulls your **App Store Connect** and **Google Play Console** statistics from one place. Instead of logging into two dashboards — or wiring up two different APIs — you get downloads, revenue, crashes, ANR rates and ratings for both iOS and Android in a single, scriptable command.

It's the cross-platform answer to the question *"is there an `app-store-connect` CLI equivalent for Google Play that also pulls stats?"* — built as a single static Go binary, so there's no Node or Python runtime to install.

## Why it exists

Most tools only cover one store, and the CLIs that exist focus on CI/CD (uploading builds), not analytics:

| Tool | Stores | Stats? | Runtime |
|---|---|---|---|
| `app-store-connect` (codemagic) | iOS | ❌ build/deploy only | Python |
| `fastlane` | iOS + Android | ❌ build/deploy only | Ruby |
| `gpc` / `playconsole-cli` | Android | ✅ Android only | TS / Go |
| **statra** | **iOS + Android** | ✅ **both, merged** | **single binary** |

statra normalizes both stores into one shape so you can chart iOS and Android downloads, revenue and crashes **side by side**.

```
Layer                     iOS                              Android
─────────────────────────────────────────────────────────────────────────────
Downloads / revenue       App Store Connect Analytics      Play monthly CSV
                          & Sales Reports API              reports (GCS bucket)
Crashes / ANR / vitals    ASC Analytics Reports            Play Developer
                                                           Reporting API
```

## Install

statra is a single static binary — **no Node, Python, or runtime required**.

```bash
# with Go (1.26+)
go install github.com/cjaryou/statra@latest

# or build from source
git clone https://github.com/cjaryou/statra
cd statra
go build -o statra .
```

## Configure

```bash
cp .env.example .env
# then fill in your Apple and/or Google credentials
```

**Apple — App Store Connect API key**
1. App Store Connect → **Users and Access → Integrations → App Store Connect API**
2. Generate a key, download the `.p8` file, note the **Issuer ID** and **Key ID**
3. Set `ASC_ISSUER_ID`, `ASC_KEY_ID`, `ASC_PRIVATE_KEY_PATH`, `ASC_APP_ID`

**Google — Play Console service account**
1. Create a service account in Google Cloud and download its JSON key
2. Grant it access in **Play Console → Users and permissions**
3. Set `GOOGLE_SERVICE_ACCOUNT_JSON`, `GOOGLE_PACKAGE_NAME`

## Usage

```bash
statra ping ios          # verify Apple credentials
statra ping android      # verify Google credentials

statra stats all --from 2026-05-01 --to 2026-05-31
statra stats ios --from 2026-05-01 --to 2026-05-31

# machine-readable output for AI agents, scripts and pipelines
statra stats ios --from 2026-05-01 --to 2026-05-31 --json
```

## Built for agents &amp; automation

statra is designed to be consumed by AI coding agents (Claude Code, etc.),
scripts and CI — not just humans:

- **`--json`** emits normalized rows agents can parse directly — no scraping tables.
- **stdout is data only**; all logs and errors go to **stderr**.
- **Non-interactive**: credentials come from env / `.env`, never prompts.
- **Stable exit codes**: non-zero on any failure.

```json
{ "from": "2026-05-01", "to": "2026-05-31",
  "rows": [ { "platform": "ios", "app": "…", "app_id": "…",
              "date": "2026-05-01", "metric": "installs", "value": 42, "unit": "count" } ] }
```

## Roadmap

- [x] Single-binary CLI (cobra)
- [x] Apple auth (ES256 JWT) + `ping` (auto-discovers your apps)
- [x] Apple Sales Reports: installs + revenue per app, app/subscription split
- [x] `--json` / `--csv` machine-readable output (agent / script friendly)
- [x] `--app` filter (by id or name)
- [x] Google auth (service account) + `ping`
- [x] Google vitals via Reporting API (crash rate)
- [ ] Google installs/revenue via GCS monthly reports
- [ ] Cross-platform merge polish + Homebrew tap (`brew install statra`)

## FAQ

**Is there a Google Play Console CLI equivalent of `app-store-connect`?**
Partly — `app-store-connect` (codemagic) is iOS CI/CD, not stats. statra is the cross-platform tool that pulls *stats* from both App Store Connect and Google Play.

**Does it scrape the dashboards?**
No. statra uses the official **App Store Connect API** and **Google Play Developer Reporting API** with your own credentials.

**Which metrics?**
Downloads, revenue, active devices, crashes, ANR (Android) and ratings — normalized across both stores.

## Contributing

Issues and PRs welcome. statra is MIT-licensed and aims to stay a lightweight, scriptable, single-binary tool.

## License

[MIT](./LICENSE)
