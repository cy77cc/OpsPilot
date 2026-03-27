import { test, expect } from '@playwright/test';
import { login } from '../support/login';

test.describe('Authentication', () => {
  test('login page displays login form', async ({ page }) => {
    await page.goto('/login');

    // Check login form elements are visible
    await expect(page.locator('input[name="username"]').or(page.locator('input[placeholder*="用户名"]'))).toBeVisible();
    await expect(page.locator('input[name="password"]').or(page.locator('input[type="password"]'))).toBeVisible();
    await expect(page.locator('button[type="submit"]').or(page.locator('button:has-text("登录")'))).toBeVisible();
  });

  test('login with invalid credentials shows error', async ({ page }) => {
    await page.goto('/login');

    // Fill in login form with invalid credentials
    const usernameInput = page.locator('input[name="username"]').or(page.locator('input[placeholder*="用户名"]'));
    const passwordInput = page.locator('input[name="password"]').or(page.locator('input[type="password"]'));

    await usernameInput.fill('invalid_user');
    await passwordInput.fill('invalid_password');

    const submitButton = page.locator('button[type="submit"]').or(page.locator('button:has-text("登录")'));
    await submitButton.click();

    // Wait for error message or stay on login page
    await page.waitForTimeout(2000);

    // Should still be on login page or show error
    const url = page.url();
    expect(url).toContain('login');
  });

  test('redirects to login when not authenticated', async ({ page }) => {
    // Try to access protected route
    await page.goto('/deployment');

    // Should redirect to login or show login prompt
    await page.waitForTimeout(1000);

    const url = page.url();
    const hasLoginPrompt = await page.locator('text=登录').isVisible().catch(() => false);

    expect(url.includes('login') || hasLoginPrompt).toBeTruthy();
  });
});

test.describe('Login helper', () => {
  test('login helper exists and is callable', async () => {
    expect(typeof login).toBe('function');
  });
});
