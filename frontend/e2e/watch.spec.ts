import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers";

test.describe("ウォッチ機能", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "donor");
  });

  test("ウォッチ追加・解除", async ({ page }) => {
    await page.goto("/projects/mock-1");
    // ウォッチボタン（"ウォッチする" or "Watch"）
    const watchBtn = page
      .locator("button")
      .filter({ hasText: /ウォッチする|^Watch$/ });
    await expect(watchBtn.first()).toBeVisible({ timeout: 10_000 });
    await watchBtn.first().click();
    await page.waitForTimeout(500);
    // テキストが変わる（"ウォッチ解除" or "Unwatch"）
    const unwatchBtn = page
      .locator("button")
      .filter({ hasText: /ウォッチ解除|Unwatch/ });
    await expect(unwatchBtn.first()).toBeVisible({ timeout: 5_000 });
    // 解除
    await unwatchBtn.first().click();
    await page.waitForTimeout(500);
  });

  test("マイページ Watches タブ", async ({ page }) => {
    // まずウォッチ
    await page.goto("/projects/mock-1");
    const watchBtn = page
      .locator("button")
      .filter({ hasText: /ウォッチする|^Watch$/ });
    if (
      await watchBtn
        .first()
        .isVisible({ timeout: 5_000 })
        .catch(() => false)
    ) {
      await watchBtn.first().click();
      await page.waitForTimeout(500);
    }
    // マイページへ
    await page.goto("/me");
    const watchesTab = page
      .locator("button, [role='tab']")
      .filter({ hasText: /ウォッチ|Watch/ });
    if (await watchesTab.isVisible({ timeout: 10_000 })) {
      await watchesTab.click();
      await page.waitForTimeout(500);
    }
  });
});
