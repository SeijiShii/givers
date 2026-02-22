import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers";

test.describe("シェアボタン", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "host");
  });

  test("シェアボタン 3 つ表示", async ({ page }) => {
    await page.goto("/projects/mock-1");
    const shareButtons = page.locator(".share-btn");
    await expect(shareButtons.first()).toBeVisible({ timeout: 10_000 });
    const count = await shareButtons.count();
    expect(count).toBe(3);
  });

  test("ダイアログ開閉", async ({ page }) => {
    await page.goto("/projects/mock-1");
    // X ボタンクリック
    const xBtn = page.locator(".share-btn").first();
    await xBtn.click();
    // ダイアログが開く
    const dialog = page.locator(".share-dialog");
    await expect(dialog).toBeVisible({ timeout: 5_000 });
    // キャンセル
    const cancelBtn = dialog.locator("button").filter({ hasText: /キャンセル|Cancel/ });
    await cancelBtn.click();
    await expect(dialog).not.toBeVisible();
  });

  test("メッセージ編集・投稿", async ({ page }) => {
    await page.goto("/projects/mock-1");
    const xBtn = page.locator(".share-btn").first();
    await xBtn.click();
    const dialog = page.locator(".share-dialog");
    await expect(dialog).toBeVisible({ timeout: 5_000 });
    // textarea にメッセージ入力
    const textarea = dialog.locator("textarea");
    await textarea.fill("カスタムシェアメッセージ");
    // window.open をインターセプト
    const openPromise = page.waitForEvent("popup", { timeout: 5_000 }).catch(() => null);
    const postBtn = dialog.locator("button").filter({ hasText: /投稿|Post/ });
    await postBtn.click();
    // ダイアログが閉じる
    await expect(dialog).not.toBeVisible();
  });

  test("localStorage にメッセージ保存", async ({ page }) => {
    await page.goto("/projects/mock-1");
    const xBtn = page.locator(".share-btn").first();
    await xBtn.click();
    const dialog = page.locator(".share-dialog");
    const textarea = dialog.locator("textarea");
    await textarea.fill("保存テスト");
    // popup を無視してボタンクリック
    page.on("popup", (popup) => popup.close());
    const postBtn = dialog.locator("button").filter({ hasText: /投稿|Post/ });
    await postBtn.click();
    // リロード後に再確認
    await page.reload();
    await page.locator(".share-btn").first().click();
    const dialog2 = page.locator(".share-dialog");
    await expect(dialog2).toBeVisible({ timeout: 5_000 });
    const saved = await dialog2.locator("textarea").inputValue();
    expect(saved).toBe("保存テスト");
  });

  test("ホームページにもシェアボタン表示", async ({ page }) => {
    await page.goto("/");
    const shareButtons = page.locator(".share-btn");
    await expect(shareButtons.first()).toBeVisible({ timeout: 10_000 });
  });
});
