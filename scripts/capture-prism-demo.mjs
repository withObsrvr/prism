import { chromium } from "playwright";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const outputDir = path.resolve("docs/demo/screenshots");
const chromiumPath = process.env.PRISM_DEMO_CHROMIUM || "/run/current-system/sw/bin/chromium";
const viewport = { width: 1600, height: 1000 };

const targets = {
  latestLedger: "https://latestledger.com/?network=mainnet",
  mainnetHealth: "https://obsrvr-lake-mainnet.withobsrvr.com/health",
  mainnetStats: "https://obsrvr-lake-mainnet.withobsrvr.com/api/v1/bronze/stats/network",
  historicalLedger: "https://obsrvr-lake-mainnet.withobsrvr.com/api/v1/bronze/ledgers?start=1000000&end=1000000&limit=1&order=asc",
  prismHome: "https://prism.withobsrvr.com/v2/home?network=testnet",
  prismLedger: "https://prism.withobsrvr.com/v2/ledger/3630463?network=testnet",
  sorobanTx: "https://prism.withobsrvr.com/v2/tx/2c35288cc669ba507a3aeacce3b3db75dee34b5b9645ef36a869f8335f037f80?network=testnet",
  classicTx: "https://prism.withobsrvr.com/v2/tx/4f4fd9bbb34ba918d6c89320a767083b99d40f7044f5296caa91d7bdec43befe?network=testnet",
  account: "https://prism.withobsrvr.com/v2/account/GA5XW2R4ALW4FLZK74Z6Z3MOBLOI2LFQ3RBZKOV2NVWCVCBNRMSJWQXH?network=testnet",
  contract: "https://prism.withobsrvr.com/v2/contract/CCLMVGEWQSKCP7HYHGDF5LPFLWNUTTWAGZPCPN5RRCD4LD5A4ZDNPPQX?network=testnet",
  swagger: "https://obsrvr-lake-testnet.withobsrvr.com/swagger/index.html",
};

const report = {
  capturedAt: new Date().toISOString(),
  viewport,
  latestLedger: {},
  captures: [],
  warnings: [],
};

await mkdir(outputDir, { recursive: true });

const browser = await chromium.launch({
  headless: true,
  executablePath: chromiumPath,
  args: ["--disable-dev-shm-usage"],
});

const context = await browser.newContext({
  viewport,
  deviceScaleFactor: 1,
  ignoreHTTPSErrors: true,
  colorScheme: "light",
});

await context.addCookies([
  {
    name: "prism_network",
    value: "testnet",
    domain: "prism.withobsrvr.com",
    path: "/",
    sameSite: "Lax",
  },
]);

const page = await context.newPage();
page.setDefaultTimeout(20_000);

async function settle(delay = 800) {
  await page.waitForLoadState("domcontentloaded");
  await page.waitForTimeout(delay);
}

async function open(url, waitUntil = "domcontentloaded") {
  const response = await page.goto(url, { waitUntil, timeout: 45_000 });
  if (!response || !response.ok()) {
    throw new Error(`GET ${url} returned ${response?.status() ?? "no response"}`);
  }
  return response;
}

async function capture(filename, description) {
  const destination = path.join(outputDir, filename);
  await page.screenshot({ path: destination, fullPage: false });
  report.captures.push({ filename, description, url: page.url(), title: await page.title() });
}

async function captureAround(locator, filename, description, offset = 90) {
  await locator.waitFor({ state: "visible" });
  await locator.evaluate((element, topOffset) => {
    const top = element.getBoundingClientRect().top + window.scrollY - topOffset;
    window.scrollTo({ top: Math.max(0, top), behavior: "instant" });
  }, offset);
  await page.waitForTimeout(400);
  await capture(filename, description);
}

async function formatJsonPage(response, heading) {
  const raw = await page.locator("body").innerText();
  let formatted = raw;
  try {
    formatted = JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    report.warnings.push(`${heading}: response was not valid JSON`);
  }

  await page.evaluate(
    ({ body, title, url, status }) => {
      document.documentElement.style.background = "#161719";
      document.body.replaceChildren();
      document.body.style.cssText = "margin:0;background:#161719;color:#f4f4f0;font-family:ui-monospace,SFMono-Regular,Menlo,monospace";

      const header = document.createElement("header");
      header.style.cssText = "position:sticky;top:0;padding:22px 34px;background:#202124;border-bottom:1px solid #3a3b3f;z-index:1";

      const label = document.createElement("div");
      label.textContent = title;
      label.style.cssText = "font:600 13px/1.4 system-ui,sans-serif;letter-spacing:.12em;text-transform:uppercase;color:#ff7f50;margin-bottom:8px";
      header.appendChild(label);

      const request = document.createElement("div");
      request.textContent = `GET ${url}`;
      request.style.cssText = "font-size:15px;color:#d8d8d4;overflow-wrap:anywhere";
      header.appendChild(request);

      const statusLabel = document.createElement("span");
      statusLabel.textContent = `HTTP ${status}`;
      statusLabel.style.cssText = "display:inline-block;margin-top:10px;padding:4px 8px;background:#173d2c;color:#81d4a3;border:1px solid #2c694b;border-radius:3px;font-size:12px";
      header.appendChild(statusLabel);

      const pre = document.createElement("pre");
      pre.textContent = body;
      pre.style.cssText = "margin:0;padding:30px 34px 60px;font-size:15px;line-height:1.55;white-space:pre-wrap;overflow-wrap:anywhere;color:#f4f4f0";

      document.body.append(header, pre);
    },
    { body: formatted, title: heading, url: response.url(), status: response.status() },
  );
  await page.waitForTimeout(200);
}

async function captureJson(url, filename, description, heading) {
  const response = await open(url);
  await formatJsonPage(response, heading);
  await capture(filename, description);
}

async function waitForLatestLedger(maxWaitMs = 90_000) {
  const deadline = Date.now() + maxWaitMs;
  let observation = {};

  do {
    await open(targets.latestLedger);
    await settle(300);
    const age = (await page.locator(".status-row .muted").textContent().catch(() => "unknown"))?.trim();
    const ageMatch = age?.match(/closed\s+(?:(\d+)m\s+)?(\d+)s\s+ago/);
    const ageSeconds = ageMatch ? Number(ageMatch[1] || 0) * 60 + Number(ageMatch[2]) : Number.POSITIVE_INFINITY;
    observation = {
      status: (await page.locator(".pill").first().textContent().catch(() => "unknown"))?.trim(),
      age,
      ageSeconds,
      ledger: (await page.locator(".ledger-number").textContent().catch(() => "unknown"))?.trim(),
      data: (await page.locator(".summary-cell").filter({ hasText: "Data" }).locator(".stat-value").textContent().catch(() => "unknown"))?.trim(),
    };

    if (observation.status === "healthy" && observation.ageSeconds < 30) break;
    await page.waitForTimeout(5_000);
  } while (Date.now() < deadline);

  report.latestLedger = observation;
  if (observation.status !== "healthy" || observation.ageSeconds >= 30) {
    report.warnings.push(`Latest Ledger did not reach a healthy under-30-second state (${observation.status}, ${observation.age}) during capture`);
  }
  if (observation.data !== "complete") {
    report.warnings.push(`Latest Ledger data completeness was ${observation.data} during capture`);
  }
}

async function waitForReceipt() {
  await page.getByText("Decoding transaction", { exact: false }).waitFor({ state: "hidden", timeout: 30_000 }).catch(() => {});
  await page.getByText("What was", { exact: false }).first().waitFor({ state: "visible", timeout: 30_000 }).catch(() => {});
  await page.waitForTimeout(1_000);
}

async function runStep(label, work) {
  try {
    await work();
  } catch (error) {
    report.warnings.push(`${label}: ${error.message}`);
  }
}

await runStep("Latest Ledger", async () => {
  await waitForLatestLedger();
  await page.evaluate(() => window.scrollTo(0, 0));
  await capture("01-latestledger-mainnet.png", "Mainnet ingestion status on Latest Ledger");
});

await runStep("Mainnet Lake health", async () => {
  await captureJson(targets.mainnetHealth, "02-mainnet-lake-health.png", "Mainnet Lake layer and index health", "Mainnet Lake health");
});

await runStep("Mainnet network stats", async () => {
  await captureJson(targets.mainnetStats, "03-mainnet-network-stats.png", "Current mainnet ledger and activity statistics", "Mainnet network stats");
});

await runStep("Historical mainnet ledger", async () => {
  await captureJson(targets.historicalLedger, "04-mainnet-ledger-1000000.png", "Historical mainnet ledger 1,000,000", "Historical ledger evidence");
});

await runStep("Prism testnet home", async () => {
  await open(targets.prismHome, "networkidle");
  await settle(500);
  await page.evaluate(() => window.scrollTo(0, 0));
  await capture("05-prism-testnet-home.png", "Prism testnet home and global search");
});

await runStep("Prism mixed ledger", async () => {
  await open(targets.prismLedger, "networkidle");
  await settle(500);
  await page.evaluate(() => window.scrollTo(0, 0));
  await capture("06-prism-ledger-overview.png", "Ledger 3,630,463 overview");
  await captureAround(page.locator("section[data-px-ledger-tx-root]"), "07-prism-ledger-classic-soroban.png", "Classic and Soroban transaction filters and rows");
});

await runStep("Soroban receipt", async () => {
  await open(targets.sorobanTx);
  await waitForReceipt();
  await page.evaluate(() => window.scrollTo(0, 0));
  await capture("08-prism-soroban-receipt.png", "Human-readable Soroban push transaction receipt");
});

await runStep("Classic receipt", async () => {
  await open(targets.classicTx);
  await waitForReceipt();
  await page.evaluate(() => window.scrollTo(0, 0));
  await capture("09-prism-classic-receipt.png", "Human-readable classic multi-operation receipt");
});

await runStep("Classic account", async () => {
  await open(targets.account);
  await page.getByText("Loading account evidence", { exact: false }).waitFor({ state: "hidden", timeout: 30_000 }).catch(() => {});
  await page.getByText("XLM balance", { exact: true }).first().waitFor({ state: "visible", timeout: 30_000 }).catch(() => {});
  await page.waitForTimeout(800);
  await page.evaluate(() => window.scrollTo(0, 0));
  await capture("10-prism-classic-account.png", "Classic account balance, reserves, and authorization evidence");
});

await runStep("Soroban contract", async () => {
  await open(targets.contract, "networkidle");
  await settle(500);
  await page.evaluate(() => window.scrollTo(0, 0));
  await capture("11-prism-contract-overview.png", "Soroban contract overview and functions");
  await page.locator('button[data-cn-tab="storage"]').click();
  await page.locator('[data-cn-panel="storage"]:visible').waitFor({ state: "visible" });
  await captureAround(page.locator("[data-stx-root]"), "12-prism-contract-storage.png", "Contract storage, TTL, and rent explorer", 140);
});

await runStep("Swagger API", async () => {
  await open(targets.swagger, "networkidle");
  await page.locator(".swagger-ui").first().waitFor({ state: "visible", timeout: 30_000 });
  await page.waitForTimeout(1_000);
  await page.evaluate(() => window.scrollTo(0, 0));
  await capture("13-testnet-swagger-overview.png", "Public Stellar Query API documentation");

  const ledgerEndpoint = page.getByText("/api/v1/silver/ledgers/recent", { exact: true }).first();
  await captureAround(ledgerEndpoint, "14-testnet-swagger-ledger-endpoint.png", "Documented recent-ledgers API endpoint", 180);
});

await browser.close();

const reportPath = path.join(outputDir, "capture-report.json");
await writeFile(reportPath, `${JSON.stringify(report, null, 2)}\n`);

const markdown = [
  "# Prism MVP Demo Screenshots",
  "",
  `Captured: ${report.capturedAt}`,
  "",
  `Latest Ledger observation: **${report.latestLedger.status ?? "unknown"}**, ${report.latestLedger.age ?? "unknown"}, ledger ${report.latestLedger.ledger ?? "unknown"}, data ${report.latestLedger.data ?? "unknown"}.`,
  "",
  "## Captures",
  "",
  ...report.captures.map((item) => `- [${item.filename}](./${item.filename}) - ${item.description}`),
  "",
  "## Warnings",
  "",
  ...(report.warnings.length ? report.warnings.map((warning) => `- ${warning}`) : ["- None"]),
  "",
];

await writeFile(path.join(outputDir, "README.md"), markdown.join("\n"));

console.log(JSON.stringify(report, null, 2));
