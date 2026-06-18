#!/usr/bin/env node
import { Command } from "commander";
import { AppStoreProvider } from "./providers/appstore.js";
import { GooglePlayProvider } from "./providers/googleplay.js";
import type { Platform } from "./types.js";

const program = new Command();

program
  .name("statra")
  .description("One CLI for App Store Connect + Google Play stats")
  .version("0.1.0");

program
  .command("ping")
  .argument("<platform>", "ios | android")
  .description("Verify credentials by reaching the store API")
  .action(async (platform: Platform) => {
    try {
      if (platform === "ios") {
        const name = await new AppStoreProvider().ping();
        console.log(`✅ App Store Connect OK — app: ${name}`);
      } else if (platform === "android") {
        const pkg = await new GooglePlayProvider().ping();
        console.log(`✅ Google Play OK — package: ${pkg}`);
      } else {
        console.error("platform must be 'ios' or 'android'");
        process.exit(1);
      }
    } catch (err) {
      console.error(`❌ ${(err as Error).message}`);
      process.exit(1);
    }
  });

program
  .command("stats")
  .argument("[platform]", "ios | android | all", "all")
  .option("--from <date>", "start date YYYY-MM-DD")
  .option("--to <date>", "end date YYYY-MM-DD")
  .description("Fetch and merge cross-platform stats")
  .action(async (_platform, _opts) => {
    console.log(
      "stats: providers are scaffolded but the report pipelines are not wired yet.\n" +
        "Next step: run `statra ping ios` / `ping android` with real credentials,\n" +
        "then we implement the analyticsReportRequests (Apple) and metric-set queries (Google).",
    );
  });

program.parseAsync(process.argv);
