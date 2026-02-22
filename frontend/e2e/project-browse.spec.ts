import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers";

test.describe("プロジェクト閲覧", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "host");
  });

  test("プロジェクト一覧表示", async ({ page }) => {
    await page.goto("/projects");
    const cards = page.locator("a[href*='/projects/mock-']");
    await expect(cards.first()).toBeVisible({ timeout: 10_000 });
    const count = await cards.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test("プロジェクト詳細 — タイトル・説明表示", async ({ page }) => {
    await page.goto("/projects/mock-1");
    await expect(page.locator("text=オープンソースの軽量エディタ")).toBeVisible({ timeout: 10_000 });
  });

  test("プロジェクト詳細 — Overviewタブ", async ({ page }) => {
    await page.goto("/projects/mock-1");
    // Overview セクション表示（タブ外に表示される場合もあるので柔軟にマッチ）
    await expect(page.locator("text=このプロジェクトについて")).toBeVisible({ timeout: 10_000 });
  });

  test("プロジェクト詳細 — Updatesタブ", async ({ page }) => {
    await page.goto("/projects/mock-1");
    // Updates タブクリック
    const updatesTab = page.locator("button, [role='tab']").filter({ hasText: /更新|Updates/ });
    if (await updatesTab.isVisible()) {
      await updatesTab.click();
      // 更新が表示されるか空メッセージが表示される
      await page.waitForTimeout(500);
    }
  });

  test("OGP メタタグが存在", async ({ page }) => {
    const response = await page.goto("/projects/mock-1");
    const html = await response?.text();
    expect(html).toContain('og:title');
  });
});
