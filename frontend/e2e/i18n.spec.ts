import { test, expect } from "@playwright/test";
import { loginAs, openMenu } from "./helpers";

test.describe("英語ロケール", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/en");
    await loginAs(page, "host");
  });

  test("英語ホーム", async ({ page }) => {
    await page.goto("/en");
    await expect(page).toHaveTitle(/GIVErS/);
    // 英語の ActivityFeed タイトル（feed.title）
    await expect(
      page.getByRole("heading", { name: "What's happening" }),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("英語プロジェクト詳細", async ({ page }) => {
    await page.goto("/en/projects/mock-1");
    await expect(page.locator("main")).toBeVisible({ timeout: 10_000 });
  });

  test("言語切替リンク", async ({ page }) => {
    await page.goto("/");
    // 英語切替リンクはナビメニュー内
    await openMenu(page);
    const enLink = page.locator("a").filter({
      hasText: /English|EN/,
    });
    await expect(enLink.first()).toBeVisible({ timeout: 10_000 });
  });
});
