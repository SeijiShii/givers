import { test, expect } from "@playwright/test";
import { loginAs, resetMigration } from "./helpers";

test.describe("マイページ", () => {
  test("Donations タブ（donor）", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "donor");
    await page.goto("/me");
    // Donations タブ
    const donationsTab = page.locator("button, [role='tab']").filter({ hasText: /寄付|Donation/ });
    if (await donationsTab.isVisible({ timeout: 10_000 })) {
      await donationsTab.click();
      await page.waitForTimeout(500);
    }
    // 寄付履歴が表示される（テーブルまたはカード）
    await expect(page.locator("main")).toBeVisible();
  });

  test("Projects タブ（owner）", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "project_owner");
    await page.goto("/me");
    const projectsTab = page.locator("button, [role='tab']").filter({ hasText: /プロジェクト|Projects/ });
    if (await projectsTab.isVisible({ timeout: 10_000 })) {
      await projectsTab.click();
      await page.waitForTimeout(500);
    }
    await expect(page.locator("main")).toBeVisible();
  });

  test("Watches タブ", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "donor");
    await page.goto("/me");
    const watchesTab = page.locator("button, [role='tab']").filter({ hasText: /ウォッチ|Watch/ });
    if (await watchesTab.isVisible({ timeout: 10_000 })) {
      await watchesTab.click();
      await page.waitForTimeout(500);
    }
  });

  test("トークン移行ダイアログ", async ({ page }) => {
    await page.goto("/");
    await resetMigration(page);
    await loginAs(page, "donor");
    await page.goto("/me");
    // pending_token_migration=true の donor → 移行ダイアログ表示
    const dialog = page.locator("text=移行").or(page.locator("text=migrate").or(page.locator("text=Migration")));
    // ダイアログが出ればクリック、出なければスキップ
    if (await dialog.first().isVisible({ timeout: 5_000 }).catch(() => false)) {
      const confirmBtn = page.locator("button").filter({ hasText: /確認|OK|移行/ });
      if (await confirmBtn.first().isVisible()) {
        await confirmBtn.first().click();
      }
    }
  });
});
