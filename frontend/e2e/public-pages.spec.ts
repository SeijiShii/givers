import { test, expect } from "@playwright/test";

test.describe("公開ページ（未ログイン）", () => {
  test.beforeEach(async ({ page }) => {
    // ログアウト状態にする
    await page.goto("/");
    await page.evaluate(() =>
      window.localStorage.setItem("givers_mock_login_mode", "logout"),
    );
    await page.reload();
  });

  test("ホームページ表示", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveTitle(/GIVErS/);
    // ActivityFeed が表示される（i18n key: feed.title）
    await expect(
      page
        .getByRole("heading", { name: "いま起きていること" })
        .or(page.getByRole("heading", { name: "What's happening" })),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("プロジェクト一覧", async ({ page }) => {
    await page.goto("/projects");
    const cards = page.locator("a[href*='/projects/mock-']");
    await expect(cards.first()).toBeVisible({ timeout: 10_000 });
  });

  test("FAQ ページ", async ({ page }) => {
    await page.goto("/faq");
    await expect(page).toHaveTitle(/FAQ|よくある質問/);
  });

  test("About ページ", async ({ page }) => {
    await page.goto("/about");
    await expect(page.locator("main")).toBeVisible();
  });

  test("利用規約", async ({ page }) => {
    await page.goto("/terms");
    await expect(page.locator("main")).toBeVisible();
  });

  test("プライバシーポリシー", async ({ page }) => {
    await page.goto("/privacy");
    await expect(page.locator("main")).toBeVisible();
  });

  test("お問い合わせフォーム送信", async ({ page }) => {
    await page.goto("/contact");
    const emailInput = page.locator("#contact-email");
    await expect(emailInput).toBeVisible({ timeout: 10_000 });
    await emailInput.fill("test@example.com");
    const messageInput = page.locator("#contact-message");
    await messageInput.fill("テストメッセージ");
    await page.locator("button[type='submit']").click();
    // 送信成功のフィードバック（contact.successTitle）
    await expect(
      page
        .getByRole("heading", { name: "送信しました" })
        .or(page.getByRole("heading", { name: "Message sent" })),
    ).toBeVisible({ timeout: 10_000 });
  });

  test("ホストページ", async ({ page }) => {
    await page.goto("/host");
    await expect(page.locator("main")).toBeVisible();
  });
});
