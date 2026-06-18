import { SignJWT, importPKCS8 } from "jose";
import { request } from "undici";
import { appleConfig, type AppleConfig } from "../config.js";
import type { MetricName, MetricRow, StatsProvider, StatsQuery } from "../types.js";

const ASC_BASE = "https://api.appstoreconnect.apple.com/v1";

/**
 * App Store Connect provider.
 *
 * Auth: ES256-signed JWT from the .p8 key (20-min expiry, per Apple spec).
 * Stats: Apple exposes downloads/revenue via the Analytics Reports API
 * (asynchronous: request a report → poll instances → download gzipped TSV).
 * Sales/finance numbers come from salesReports (synchronous gzipped TSV).
 */
export class AppStoreProvider implements StatsProvider {
  readonly platform = "ios" as const;
  private cfg: AppleConfig;

  constructor(cfg: AppleConfig = appleConfig()) {
    this.cfg = cfg;
  }

  /** Generate a short-lived bearer token signed with the App Store Connect key. */
  private async token(): Promise<string> {
    const key = await importPKCS8(this.cfg.privateKey, "ES256");
    const now = Math.floor(Date.now() / 1000);
    return new SignJWT({})
      .setProtectedHeader({ alg: "ES256", kid: this.cfg.keyId, typ: "JWT" })
      .setIssuer(this.cfg.issuerId)
      .setIssuedAt(now)
      .setExpirationTime(now + 20 * 60)
      .setAudience("appstoreconnect-v1")
      .sign(key);
  }

  private async authedGet(path: string, query: Record<string, string> = {}): Promise<any> {
    const url = new URL(`${ASC_BASE}${path}`);
    for (const [k, v] of Object.entries(query)) url.searchParams.set(k, v);
    const res = await request(url, {
      method: "GET",
      headers: { authorization: `Bearer ${await this.token()}` },
    });
    if (res.statusCode >= 400) {
      throw new Error(`ASC ${res.statusCode}: ${await res.body.text()}`);
    }
    return res.body.json();
  }

  /** Quick credential check — lists the configured app. */
  async ping(): Promise<string> {
    const data = await this.authedGet(`/apps/${this.cfg.appId}`);
    return data?.data?.attributes?.name ?? this.cfg.appId;
  }

  async fetch(query: StatsQuery, metrics: MetricName[]): Promise<MetricRow[]> {
    // The Analytics Reports flow is async (request → poll → download TSV).
    // Wired next: POST /analyticsReportRequests, poll instances, gunzip+parse.
    void query;
    void metrics;
    throw new Error(
      "AppStoreProvider.fetch: report pipeline not wired yet — run `statra ping ios` first to confirm auth, then we implement analyticsReportRequests.",
    );
  }
}
