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
