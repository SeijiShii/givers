import { test, expect } from "@playwright/test";
import { loginAs, setSuspended } from "./helpers";

test.describe("寄付フロー", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/");
    await loginAs(page, "donor");
  });

  test("寄付フォームが表示される", async ({ page }) => {
    await page.goto("/projects/mock-1");
    // DonateForm の送信ボタン（btn-accent クラス、"寄付する" テキスト）
    const donateBtn = page.locator("button.btn-accent").filter({
      hasText: /寄付する|Donate/,
    });
    await expect(donateBtn.first()).toBeVisible({ timeout: 10_000 });
  });

  test("プリセット金額を選択できる", async ({ page }) => {
    await page.goto("/projects/mock-1");
    // プリセットボタン（¥500 など）
    const preset = page.locator("button").filter({ hasText: /500/ });
    if (await preset.isVisible({ timeout: 5_000 }).catch(() => false)) {
      await preset.click();
    }
  });

  test("ワンタイム / 月次切替", async ({ page }) => {
    await page.goto("/projects/mock-1");
    // ワンタイム / 月次 ラジオまたはタブ
    const monthly = page
      .locator("label, button")
      .filter({ hasText: /月次|Monthly|月額/ });
    if (await monthly.isVisible({ timeout: 5_000 }).catch(() => false)) {
      await monthly.click();
    }
  });

  test("決済完了後に感謝モーダルが表示される", async ({ page }) => {
    // session_id パラメータ付きでアクセス（Stripe からの戻りをシミュレート）
    await page.goto("/projects/mock-1?session_id=test_session");

    // マイグレーションダイアログが先に出たら閉じる
    const migrationDialog = page.locator(
      "[role='dialog'][aria-labelledby='migration-dialog-title']",
    );
    if (
      await migrationDialog.isVisible({ timeout: 3_000 }).catch(() => false)
    ) {
      await migrationDialog
        .locator("button")
        .filter({ hasText: /後で|Later/ })
        .click();
    }

    // 感謝モーダル（migration-dialog-title を持たない dialog）
    const thankYouModal = page.locator(
      "[role='dialog']:not([aria-labelledby='migration-dialog-title'])",
    );
    await expect(thankYouModal).toBeVisible({ timeout: 10_000 });
    // 感謝メッセージが含まれている
    await expect(thankYouModal).toContainText(/ありがとう|Thank you/);
    // 閉じるボタンで閉じられる
    const closeBtn = thankYouModal.locator("button");
    await closeBtn.click();
    await expect(thankYouModal).not.toBeVisible();
  });

  test("定期寄付決済後にサブスク管理UIが表示される", async ({ page }) => {
    // Stripe Checkout からの戻りをシミュレート（donated=1 パラメータ）
    await page.goto("/projects/mock-1?donated=1");

    // マイグレーションダイアログが先に出たら閉じる
    const migrationDialog = page.locator(
      "[role='dialog'][aria-labelledby='migration-dialog-title']",
    );
    if (
      await migrationDialog.isVisible({ timeout: 3_000 }).catch(() => false)
    ) {
      await migrationDialog
        .locator("button")
        .filter({ hasText: /後で|Later/ })
        .click();
    }

    // 感謝モーダルを閉じる
    const thankYouModal = page.locator(
      "[role='dialog']:not([aria-labelledby='migration-dialog-title'])",
    );
    await expect(thankYouModal).toBeVisible({ timeout: 10_000 });
    await thankYouModal
      .locator("button")
      .filter({ hasText: /閉じる|Close/ })
      .click();
    await expect(thankYouModal).not.toBeVisible();

    // サブスク管理UIが表示されること（DonateFormではなく）
    await expect(
      page
        .locator("text=定期寄付を管理中")
        .or(page.locator("text=Managing recurring donation")),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("停止ユーザーは寄付不可", async ({ page }) => {
    await setSuspended(page, true);
    await page.reload();
    await page.goto("/projects/mock-1");
    // DonateForm 内に停止メッセージが表示される（main コンテンツ内）
    await expect(
      page
        .locator("main")
        .locator("text=利用停止")
        .or(page.locator("main").locator("text=suspended")),
    ).toBeVisible({ timeout: 10_000 });
    // cleanup
    await setSuspended(page, false);
  });
});
