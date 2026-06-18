# statra

> One CLI for **App Store Connect + Google Play** stats — downloads, revenue, crashes, ratings in a single place.

Most tools do one store. `app-store-connect` (codemagic) handles iOS CI/CD; `gpc` / `playconsole-cli` handle Android. None of them give you **iOS and Android stats merged in one command**. That's the gap statra fills.

## Why

| Layer | iOS | Android |
|---|---|---|
| Downloads / revenue | App Store Connect — Analytics & Sales Reports API | Play monthly CSV reports (GCS bucket) |
| Crashes / ANR / vitals | ASC Analytics Reports | Play Developer Reporting API |

statra normalizes both into one shape so you can chart them together.

## Install

statra is a single static binary — no Node, Python, or runtime required.

```bash
# from source (Go 1.26+)
go install github.com/cjaryou/statra@latest

# or build locally
git clone https://github.com/cjaryou/statra
cd statra
go build -o statra .
```

## Configure

```bash
cp .env.example .env
# fill in Apple (.p8 key) + Google (service-account.json) credentials
```

- **Apple:** App Store Connect → Users and Access → Integrations → App Store Connect API → generate key (`.p8`). Note Issuer ID + Key ID.
- **Google:** Create a Google Cloud service account, grant it access in Play Console → Users and permissions, download the JSON key.

## Usage

```bash
statra ping ios          # verify Apple credentials
statra ping android      # verify Google credentials
statra stats all --from 2026-05-01 --to 2026-05-31
```

## Built with

Go — compiles to a single static binary for easy distribution (`brew` / `go install`),
the same approach as `gh`, `terraform`, and the leading Play Console CLIs.

## Status

- [x] Project scaffold + CLI (cobra)
- [x] Apple auth (ES256 JWT) + `ping`
- [x] Google auth (service account) + `ping`
- [ ] Apple Analytics/Sales report pipeline
- [ ] Google metric-set queries + GCS CSV download
- [ ] Cross-platform merge + output formats (table / json / csv)

## License

MIT
