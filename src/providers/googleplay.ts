import { GoogleAuth } from "google-auth-library";
import { googleConfig, type GoogleConfig } from "../config.js";
import type { MetricName, MetricRow, StatsProvider, StatsQuery } from "../types.js";

const REPORTING_BASE = "https://playdeveloperreporting.googleapis.com/v1beta1";

/**
 * Google Play provider.
 *
 * Auth: service-account JWT → OAuth2 access token (google-auth-library).
 * Vitals (crashes, ANR, slow start, active devices) come from the
 * Play Developer Reporting API metric sets. Downloads/revenue come from the
 * monthly CSV reports Play writes to a GCS bucket.
 */
export class GooglePlayProvider implements StatsProvider {
  readonly platform = "android" as const;
  private cfg: GoogleConfig;
  private auth: GoogleAuth;

  constructor(cfg: GoogleConfig = googleConfig()) {
    this.cfg = cfg;
    this.auth = new GoogleAuth({
      credentials: {
        client_email: cfg.serviceAccount.client_email,
        private_key: cfg.serviceAccount.private_key,
      },
      scopes: ["https://www.googleapis.com/auth/playdeveloperreporting"],
    });
  }

  private async token(): Promise<string> {
    const client = await this.auth.getClient();
    const t = await client.getAccessToken();
    if (!t.token) throw new Error("Failed to obtain Google access token");
    return t.token;
  }

  /** Quick credential check — reads the app's reporting freshness info. */
  async ping(): Promise<string> {
    const token = await this.token();
    const res = await fetch(
      `${REPORTING_BASE}/apps/${this.cfg.packageName}:fetchReleaseFilterOptions`,
      { method: "POST", headers: { authorization: `Bearer ${token}` } },
    );
    if (!res.ok) throw new Error(`Reporting API ${res.status}: ${await res.text()}`);
    return this.cfg.packageName;
  }

  async fetch(query: StatsQuery, metrics: MetricName[]): Promise<MetricRow[]> {
    // Wired next: POST /vitals/crashrate:query etc. for the metric sets,
    // plus GCS CSV download for installs/revenue.
    void query;
    void metrics;
    throw new Error(
      "GooglePlayProvider.fetch: metric-set queries not wired yet — run `statra ping android` first to confirm auth, then we implement the crashrate/errors queries.",
    );
  }
}
