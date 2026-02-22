import { type Page } from "@playwright/test";

const MOCK_LOGIN_MODE_KEY = "givers_mock_login_mode";
const MOCK_SUSPENDED_KEY = "givers_mock_suspended_user";
const MOCK_MIGRATION_DONE_KEY = "givers_mock_migration_done";

export type MockRole = "host" | "project_owner" | "donor" | "logout";

/** モックモードでロール切替 → ページリロード */
export async function loginAs(page: Page, role: MockRole) {
  await page.evaluate(
    ([key, val]) => window.localStorage.setItem(key, val),
    [MOCK_LOGIN_MODE_KEY, role],
  );
  await page.reload();
  await page.waitForLoadState("networkidle");
}

/** ログアウト */
export async function logout(page: Page) {
  await loginAs(page, "logout");
}

/** ハンバーガーメニューを開く */
export async function openMenu(page: Page) {
  const toggle = page.locator("[data-nav-toggle]");
  await toggle.click();
  await page.locator(".nav-menu.is-open").waitFor({ timeout: 5_000 });
}

/** suspended フラグをセット */
export async function setSuspended(page: Page, suspended: boolean) {
  await page.evaluate(
    ([key, val]) => {
      if (val === "true") window.localStorage.setItem(key, val);
      else window.localStorage.removeItem(key);
    },
    [MOCK_SUSPENDED_KEY, suspended ? "true" : "false"],
  );
}

/** token migration done フラグをリセット */
export async function resetMigration(page: Page) {
  await page.evaluate(
    (key) => window.localStorage.removeItem(key),
    MOCK_MIGRATION_DONE_KEY,
  );
}
