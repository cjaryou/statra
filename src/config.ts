import { readFileSync } from "node:fs";
import { resolve } from "node:path";

/**
 * Loads credentials from environment variables (see .env.example).
 * Kept dependency-free: read a .env manually if present so users don't
 * need a runtime dotenv dependency.
 */
function loadDotEnv(): void {
  try {
    const raw = readFileSync(resolve(process.cwd(), ".env"), "utf8");
    for (const line of raw.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) continue;
      const eq = trimmed.indexOf("=");
      if (eq === -1) continue;
      const key = trimmed.slice(0, eq).trim();
      const val = trimmed.slice(eq + 1).trim();
      if (!(key in process.env)) process.env[key] = val;
    }
  } catch {
    // no .env file — rely on real environment
  }
}
loadDotEnv();

function required(name: string): string {
  const v = process.env[name];
  if (!v) throw new Error(`Missing required env var: ${name} (see .env.example)`);
  return v;
}

export interface AppleConfig {
  issuerId: string;
  keyId: string;
  privateKey: string;
  appId: string;
}

export interface GoogleConfig {
  serviceAccount: { client_email: string; private_key: string; [k: string]: unknown };
  packageName: string;
  reportsBucket?: string;
}

export function appleConfig(): AppleConfig {
  return {
    issuerId: required("ASC_ISSUER_ID"),
    keyId: required("ASC_KEY_ID"),
    privateKey: readFileSync(resolve(required("ASC_PRIVATE_KEY_PATH")), "utf8"),
    appId: required("ASC_APP_ID"),
  };
}

export function googleConfig(): GoogleConfig {
  const saPath = required("GOOGLE_SERVICE_ACCOUNT_JSON");
  return {
    serviceAccount: JSON.parse(readFileSync(resolve(saPath), "utf8")),
    packageName: required("GOOGLE_PACKAGE_NAME"),
    reportsBucket: process.env.GOOGLE_REPORTS_BUCKET,
  };
}
