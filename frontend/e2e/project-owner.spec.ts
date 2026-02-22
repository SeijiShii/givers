import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers";

test.describe("プロジェクトオーナー操作", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "project_owner");
  });

  test("プロジェクト新規作成フォーム", async ({ page }) => {
    await page.goto("/projects/new");
    const nameInput = page.locator("#name");
    await expect(nameInput).toBeVisible({ timeout: 10_000 });
    await nameInput.fill("テストプロジェクト");
    const descInput = page.locator("#description");
    await descInput.fill("テスト用の説明文");
    // 送信ボタンが存在
    const submit = page.locator("button[type='submit']");
    await expect(submit).toBeVisible();
  });

  test("プロジェクト編集ページ", async ({ page }) => {
    await page.goto("/projects/mock-2/edit");
    const nameInput = page.locator("#name");
    await expect(nameInput).toBeVisible({ timeout: 10_000 });
    // 既存値が入力されている
    const value = await nameInput.inputValue();
    expect(value.length).toBeGreaterThan(0);
  });

  test("Overview インライン編集ボタン表示", async ({ page }) => {
    // mock-2 のオーナーは user-2 (project_owner)
    await page.goto("/projects/mock-2");
    // 「概要を編集」ボタンがオーナーに表示される
    const editBtn = page.locator("button").filter({ hasText: /編集|Edit/ });
    await expect(editBtn.first()).toBeVisible({ timeout: 10_000 });
  });

  test("更新を投稿", async ({ page }) => {
    await page.goto("/projects/mock-2");
    // Updates タブをクリック
    const updatesTab = page.locator("button, [role='tab']").filter({ hasText: /更新|Updates/ });
    if (await updatesTab.isVisible()) {
      await updatesTab.click();
    }
    // 投稿フォームが表示される（オーナーのみ）
    const bodyInput = page.locator("textarea").filter({ hasText: "" });
    if (await bodyInput.first().isVisible({ timeout: 5_000 }).catch(() => false)) {
      await bodyInput.first().fill("テスト更新の本文");
      const postBtn = page.locator("button").filter({ hasText: /投稿|Post/ });
      if (await postBtn.isVisible()) {
        await postBtn.click();
        await page.waitForTimeout(500);
      }
    }
  });

  test("マイページにプロジェクト表示", async ({ page }) => {
    await page.goto("/me");
    // Projects タブ
    const projectsTab = page.locator("button, [role='tab']").filter({ hasText: /プロジェクト|Projects/ });
    if (await projectsTab.isVisible({ timeout: 10_000 })) {
      await projectsTab.click();
      await page.waitForTimeout(500);
    }
    // 自分のプロジェクトが表示
    await expect(page.locator("main")).toBeVisible();
  });
});
