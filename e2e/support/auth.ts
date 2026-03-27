import { test as base, Page, BrowserContext } from '@playwright/test';

// Define test user credentials
export interface TestUser {
  username: string;
  password: string;
}

// Default test user from environment or fallback
export const getTestUser = (): TestUser => ({
  username: process.env.E2E_USERNAME || 'admin',
  password: process.env.E2E_PASSWORD || 'admin123',
});

// API base URL
export const getApiBase = (): string => process.env.E2E_API_URL || 'http://localhost:8888';

// Extended test fixture with authentication
type AuthFixtures = {
  authenticatedPage: Page;
  authToken: string;
};

export const test = base.extend<AuthFixtures>({
  authenticatedPage: async ({ page }, use) => {
    const user = getTestUser();
    const apiBase = getApiBase();

    // Login via API to get token
    const response = await page.request.post(`${apiBase}/api/v1/user/login`, {
      data: {
        username: user.username,
        password: user.password,
      },
    });

    if (response.ok()) {
      const data = await response.json();
      const token = data.data?.accessToken;

      if (token) {
        // Set auth token in localStorage and cookies
        await page.addInitScript((token) => {
          localStorage.setItem('token', token);
          localStorage.setItem('accessToken', token);
        }, token);

        // Also set as cookie for API calls
        await page.context().addCookies([
          {
            name: 'Authorization',
            value: `Bearer ${token}`,
            domain: 'localhost',
            path: '/',
          },
        ]);
      }
    }

    await use(page);
  },

  authToken: async ({ page }, use) => {
    const user = getTestUser();
    const apiBase = getApiBase();

    let token = '';

    const response = await page.request.post(`${apiBase}/api/v1/user/login`, {
      data: {
        username: user.username,
        password: user.password,
      },
    });

    if (response.ok()) {
      const data = await response.json();
      token = data.data?.accessToken || '';
    }

    await use(token);
  },
});

export { expect } from '@playwright/test';

/**
 * Login via UI (for tests that need to test the login flow itself)
 */
export async function loginViaUI(page: Page, username?: string, password?: string): Promise<boolean> {
  const user = getTestUser();
  const creds = {
    username: username || user.username,
    password: password || user.password,
  };

  await page.goto('/login');
  await page.waitForLoadState('networkidle');

  // Find and fill username
  const usernameInput = page.locator('input[name="username"]').or(
    page.locator('input[placeholder*="用户名"]')
  ).or(
    page.locator('input[type="text"]').first()
  );

  // Find and fill password
  const passwordInput = page.locator('input[name="password"]').or(
    page.locator('input[type="password"]')
  );

  await usernameInput.fill(creds.username);
  await passwordInput.fill(creds.password);

  // Submit
  const submitButton = page.locator('button[type="submit"]').or(
    page.locator('button:has-text("登录")')
  );

  await submitButton.click();

  // Wait for redirect or error
  try {
    await page.waitForURL('**/dashboard**', { timeout: 10000 });
    return true;
  } catch {
    // Check if still on login page
    return !page.url().includes('login');
  }
}

/**
 * Logout helper
 */
export async function logout(page: Page): Promise<void> {
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

  await page.waitForURL('**/login**', { timeout: 5000 }).catch(() => {});
}

/**
 * Check if currently logged in
 */
export async function isLoggedIn(page: Page): Promise<boolean> {
  if (page.url().includes('login')) {
    return false;
  }

  const token = await page.evaluate(() => localStorage.getItem('token'));
  return !!token;
}
