import { strict as assert } from "node:assert";
import { existsSync } from "node:fs";
import { chromium } from "playwright";

function resolveChromiumPath() {
  if (process.env.PRISM_TEST_CHROMIUM) return process.env.PRISM_TEST_CHROMIUM;
  try {
    if (existsSync(chromium.executablePath())) return "";
  } catch {
    // Fall through to system candidates.
  }
  const candidates = [
    "/run/current-system/sw/bin/chromium",
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
    "/usr/bin/google-chrome",
  ];
  return candidates.find((candidate) => existsSync(candidate)) ?? "";
}

const baseURL = process.env.PRISM_BASE_URL ?? "http://127.0.0.1:3002";
const chromiumPath = resolveChromiumPath();
const browser = await chromium.launch({
  headless: true,
  ...(chromiumPath ? { executablePath: chromiumPath } : {}),
  args: ["--disable-dev-shm-usage"],
});

try {
  const page = await browser.newPage({ viewport: { width: 1366, height: 800 } });
  await page.goto(`${baseURL}/v2/home?mock=true&network=mainnet`, { waitUntil: "domcontentloaded" });
  await page.locator(".ph-spectrum-column").first().waitFor();

  const geometry = await page.evaluate(() => {
    const rect = (element) => {
      const value = element.getBoundingClientRect();
      return { top: value.top, bottom: value.bottom, width: value.width, height: value.height };
    };
    const columns = [...document.querySelectorAll(".ph-spectrum-column")];
    const column = columns.find((candidate) => Number.parseFloat(getComputedStyle(candidate).getPropertyValue("--ph-failure-height")) > 0);
    assertElement(column, "a ledger column with failed operations");
    const stack = column.querySelector(".ph-spectrum-stack");
    const failure = column.querySelector(".ph-spectrum-failure");
    const wrap = document.querySelector(".ph-spectrum-chart-wrap");
    assertElement(stack, "activity stack");
    assertElement(failure, "failure marker");
    assertElement(wrap, "chart wrapper");
    const stackRect = rect(stack);
    const failureRect = rect(failure);
    return {
      columnCount: columns.length,
      stackBorderRadius: getComputedStyle(stack).borderRadius,
      stack: stackRect,
      failure: failureRect,
      failureOverlapsStack: failureRect.top < stackRect.bottom && failureRect.bottom > stackRect.top,
      wrapperClientWidth: wrap.clientWidth,
      wrapperScrollWidth: wrap.scrollWidth,
    };

    function assertElement(element, label) {
      if (!element) throw new Error(`Missing ${label}`);
    }
  });

  assert.equal(geometry.columnCount, 60, "mock fixture should render 60 ledger columns");
  assert.equal(geometry.stackBorderRadius, "0px", "activity columns should be square");
  assert.equal(geometry.failureOverlapsStack, false, "failure markers must not overlap activity stacks");
  assert.ok(geometry.failure.top >= geometry.stack.bottom + 3, "failure markers need a visible gap below the activity baseline");
  assert.ok(geometry.wrapperScrollWidth <= geometry.wrapperClientWidth + 1, "desktop tooltips must not force horizontal overflow");
  console.log(JSON.stringify(geometry, null, 2));
} finally {
  await browser.close();
}
