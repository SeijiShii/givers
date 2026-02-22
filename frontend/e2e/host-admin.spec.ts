import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers";

test.describe("ホスト管理画面", () => {
  test("ユーザー管理ページ表示", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "host");
    await page.goto("/admin/users");
    // ユーザー一覧テーブル内のセル（nav-menu 内の auth-user と区別する）
    await expect(
      page.getByRole("cell", { name: "山田太郎" }),
    ).toBeVisible({ timeout: 10_000 });
    await expect(
      page.getByRole("cell", { name: "佐藤花子" }),
    ).toBeVisible();
  });

  test("コンタクト一覧表示", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "host");
    await page.goto("/host/contacts");
    await expect(page.locator("main")).toBeVisible({ timeout: 10_000 });
  });

  test("非ホストはアクセス不可", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "donor");
    await page.goto("/admin/users");
    // forbidden メッセージまたはリダイレクト
    const forbidden = page
      .locator("text=権限がありません")
      .or(
        page
          .locator("text=forbidden")
          .or(page.locator("text=Forbidden")),
      );
    await expect(forbidden.first()).toBeVisible({ timeout: 10_000 });
  });

  test("開示エクスポートボタン存在", async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "host");
    await page.goto("/admin/users");
    const exportBtn = page
      .locator("button")
      .filter({ hasText: /開示|Export|エクスポート/ });
    await expect(exportBtn.first()).toBeVisible({ timeout: 10_000 });
  });
});
