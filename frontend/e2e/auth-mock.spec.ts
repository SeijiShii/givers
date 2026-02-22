import { test, expect } from "@playwright/test";
import { loginAs, logout, openMenu } from "./helpers";

test.describe("認証（モックモード切替）", () => {
  test("ホストとしてログイン", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "host");
    await openMenu(page);
    await expect(page.locator(".nav-menu >> text=山田太郎")).toBeVisible({
      timeout: 10_000,
    });
  });

  test("プロジェクトオーナーに切替", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "project_owner");
    await openMenu(page);
    await expect(page.locator(".nav-menu >> text=佐藤花子")).toBeVisible({
      timeout: 10_000,
    });
  });

  test("寄付者に切替", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "donor");
    await openMenu(page);
    await expect(page.locator(".nav-menu >> text=高橋健太")).toBeVisible({
      timeout: 10_000,
    });
  });

  test("ログアウト", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "host");
    await openMenu(page);
    await expect(page.locator(".nav-menu >> text=山田太郎")).toBeVisible({
      timeout: 10_000,
    });
    // logout reloads the page, so no need to close menu first
    await logout(page);
    await openMenu(page);
    await expect(page.locator(".nav-menu >> text=山田太郎")).not.toBeVisible();
  });
});
