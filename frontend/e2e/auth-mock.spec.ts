import { test, expect } from "@playwright/test";
import { loginAs, logout, resetMigration } from "./helpers";

test.describe("認証（モックモード切替）", () => {
  test("ホストとしてログイン", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "host");
    await expect(
      page.locator(".auth-status-header >> text=山田太郎"),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("プロジェクトオーナーに切替", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "project_owner");
    await expect(
      page.locator(".auth-status-header >> text=佐藤花子"),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("寄付者に切替", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "donor");
    await expect(
      page.locator(".auth-status-header >> text=高橋健太"),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("ログアウト", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "host");
    await expect(
      page.locator(".auth-status-header >> text=山田太郎"),
    ).toBeVisible({ timeout: 10_000 });
    await logout(page);
    await expect(
      page.locator(".auth-status-header >> text=山田太郎"),
    ).not.toBeVisible();
  });

  test("プロジェクト詳細ページでログアウト → ホームにリダイレクト", async ({
    page,
  }) => {
    await page.goto("/");
    await loginAs(page, "donor");
    await expect(
      page.locator(".auth-status-header >> text=高橋健太"),
    ).toBeVisible({ timeout: 10_000 });

    // プロジェクト詳細ページに遷移
    await page.goto("/projects/mock-1");
    await page.waitForLoadState("networkidle");

    // マイグレーションダイアログが出たら閉じる
    const dismissBtn = page.locator("[role='dialog'] button").filter({
      hasText: /後で|Later/,
    });
    if (await dismissBtn.isVisible({ timeout: 3_000 }).catch(() => false)) {
      await dismissBtn.click();
    }

    // ログアウトボタンをクリック（ヘッダー内）
    const logoutBtn = page
      .locator(".auth-status-header button")
      .filter({ hasText: /ログアウト|Logout/ });
    await logoutBtn.click();

    // ホームにリダイレクトされること
    await page.waitForURL("/", { timeout: 10_000 });
    expect(page.url()).toMatch(/\/$/);
  });
});
