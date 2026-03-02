import { Page } from '@playwright/test';

/**
 * Login helper for E2E tests.
 * Use this to authenticate before running tests that require login.
 *
 * @param page - Playwright page object
 * @param username - Username to login with (default: from env or 'admin')
 * @param password - Password to login with (default: from env or 'admin123')
 */
export async function login(
  page: Page,
  username: string = process.env.E2E_USERNAME || 'admin',
  password: string = process.env.E2E_PASSWORD || 'admin123'
): Promise<void> {
  await page.goto('/login');

  // Wait for login form to be ready
  await page.waitForLoadState('networkidle');

  // Find username input
  const usernameInput = page.locator('input[name="username"]').or(
    page.locator('input[placeholder*="用户名"]')
  ).or(
    page.locator('input[type="text"]').first()
  );

  // Find password input
  const passwordInput = page.locator('input[name="password"]').or(
    page.locator('input[type="password"]')
  );

  // Fill credentials
  await usernameInput.fill(username);
  await passwordInput.fill(password);

  // Find and click submit button
  const submitButton = page.locator('button[type="submit"]').or(
    page.locator('button:has-text("登录")')
  ).or(
    page.locator('button:has-text("Login")')
  );

  await submitButton.click();

  // Wait for navigation or error
  await page.waitForURL('**/dashboard**', { timeout: 10000 }).catch(() => {
    // If redirect to dashboard fails, check for error message
    console.log('Login may have failed or redirected elsewhere');
  });
}

/**
 * Logout helper for E2E tests.
 *
 * @param page - Playwright page object
 */
export async function logout(page: Page): Promise<void> {
  // Find user menu or logout button
  const userMenu = page.locator('[data-testid="user-menu"]').or(
    page.locator('.anticon-user').or(page.locator('[aria-label*="user"]'))
  );

  await userMenu.click();

  const logoutButton = page.locator('button:has-text("退出")').or(
    page.locator('button:has-text("Logout")')
  ).or(
    page.locator('a:has-text("退出")')
  );

  await logoutButton.click();

  // Wait for redirect to login
  await page.waitForURL('**/login**', { timeout: 5000 }).catch(() => {
    console.log('Logout redirect may have failed');
  });
}

/**
 * Check if user is logged in.
 *
 * @param page - Playwright page object
 */
export async function isLoggedIn(page: Page): Promise<boolean> {
  const url = page.url();
  if (url.includes('login')) {
    return false;
  }

  // Check for user-related elements that indicate logged-in state
  const userMenu = page.locator('[data-testid="user-menu"]').or(
    page.locator('.anticon-user')
  );

  return userMenu.isVisible().catch(() => false);
}
