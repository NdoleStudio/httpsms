const { chromium } = require("playwright");
const path = require("path");

const LOCAL = "http://localhost:3000";
const PROD = "https://httpsms.com";

const pages = [
  { route: "/", name: "homepage" },
  { route: "/login", name: "login" },
  { route: "/privacy-policy", name: "privacy-policy" },
  { route: "/terms-and-conditions", name: "terms-and-conditions" },
  { route: "/blog", name: "blog" },
  { route: "/blog/how-to-get-unlimited-sms-api", name: "blog-article" },
];

const screenshotDir = path.join(__dirname, "screenshots");

(async () => {
  const browser = await chromium.launch({ headless: true });

  // Screenshot production site
  console.log("📸 Taking screenshots of PRODUCTION (httpsms.com)...");
  const prodContext = await browser.newContext({
    viewport: { width: 1280, height: 900 },
  });
  for (const { route, name } of pages) {
    const page = await prodContext.newPage();
    try {
      await page.goto(PROD + route, {
        waitUntil: "networkidle",
        timeout: 30000,
      });
      await page.waitForTimeout(2000);
      await page.screenshot({
        path: path.join(screenshotDir, "old", `${name}.png`),
        fullPage: true,
      });
      console.log(`  ✅ old/${name}.png`);
    } catch (err) {
      console.log(`  ❌ old/${name}.png - ${err.message.split("\n")[0]}`);
    }
    await page.close();
  }
  await prodContext.close();

  // Screenshot local dev - wait longer for SPA rendering
  console.log("\n📸 Taking screenshots of LOCAL (localhost:3000)...");
  const localContext = await browser.newContext({
    viewport: { width: 1280, height: 900 },
  });
  for (const { route, name } of pages) {
    const page = await localContext.newPage();
    try {
      await page.goto(LOCAL + route, { waitUntil: "load", timeout: 30000 });
      // Wait for Vue/Vuetify to mount - SPA needs more time
      await page.waitForSelector(".v-application", { timeout: 15000 });
      await page.waitForTimeout(3000); // Extra time for images/fonts
      await page.screenshot({
        path: path.join(screenshotDir, "new", `${name}.png`),
        fullPage: true,
      });
      console.log(`  ✅ new/${name}.png`);
    } catch (err) {
      console.log(`  ❌ new/${name}.png - ${err.message.split("\n")[0]}`);
      // Take screenshot anyway to see what's there
      await page.screenshot({
        path: path.join(screenshotDir, "new", `${name}.png`),
        fullPage: true,
      });
    }
    await page.close();
  }
  await localContext.close();

  await browser.close();
  console.log("\n✅ Done!");
})();
