import { test, expect } from "@playwright/test";
import { loginAs, logout } from "./helpers";

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
});
