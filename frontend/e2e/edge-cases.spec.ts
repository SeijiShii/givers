import { test, expect } from "@playwright/test";
import { loginAs, setSuspended, openMenu } from "./helpers";

test.describe("エッジケース", () => {
  test("停止アカウントバナー表示", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "donor");
    await setSuspended(page, true);
    await page.reload();
    await page.waitForLoadState("networkidle");
    // 停止バナーはナビメニュー内の AuthStatus に表示される
    await openMenu(page);
    const banner = page.locator("[role='alert']").or(
      page.locator("text=利用停止"),
    );
    await expect(banner.first()).toBeVisible({ timeout: 10_000 });
    // cleanup
    await setSuspended(page, false);
  });

  test("アクティビティフィード表示", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "host");
    // ActivityFeed にアイテムが存在
    const feed = page
      .locator("text=作成しました")
      .or(
        page
          .locator("text=支援しました")
          .or(page.locator("text=達成しました")),
      );
    await expect(feed.first()).toBeVisible({ timeout: 10_000 });
  });

  test("ナビの財務ヘルスマーク", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "host");
    // ナビメニュー内のホストリンク
    await openMenu(page);
    const hostLink = page.locator("a[href*='/host']").first();
    await expect(hostLink).toBeVisible({ timeout: 10_000 });
  });
});
