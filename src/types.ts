export type Platform = "ios" | "android";

export interface StatsQuery {
  /** Inclusive start date, YYYY-MM-DD */
  from: string;
  /** Inclusive end date, YYYY-MM-DD */
  to: string;
}

/** Normalized cross-platform metric row so iOS + Android can be merged. */
export interface MetricRow {
  platform: Platform;
  date: string; // YYYY-MM-DD
  metric: MetricName;
  value: number;
  unit?: string; // e.g. "USD", "count"
}

export type MetricName =
  | "installs"
  | "uninstalls"
  | "active_devices"
  | "crashes"
  | "anr" // android only
  | "rating"
  | "revenue";

export interface StatsProvider {
  readonly platform: Platform;
  fetch(query: StatsQuery, metrics: MetricName[]): Promise<MetricRow[]>;
}
