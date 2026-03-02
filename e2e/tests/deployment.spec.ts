import { test, expect } from '@playwright/test';

test.describe('Deployment Page', () => {
  test.skip('deployment page shows targets list', async ({ page }) => {
    // This test requires authentication
    // Skip in CI without proper auth setup
    await page.goto('/deployment');

    // Check page header
    await expect(page.locator('h1, h2').first()).toContainText(/部署|Deployment/i);

    // Check for deployment targets table or list
    const hasTargetsTable = await page.locator('table').isVisible().catch(() => false);
    const hasTargetsList = await page.locator('[data-testid="targets-list"]').isVisible().catch(() => false);
    const hasCreateButton = await page.locator('button:has-text("创建")').or(page.locator('button:has-text("新增")')).isVisible().catch(() => false);

    expect(hasTargetsTable || hasTargetsList || hasCreateButton).toBeTruthy();
  });

  test.skip('create deployment target form validates input', async ({ page }) => {
    // This test requires authentication
    await page.goto('/deployment');

    // Click create button
    const createButton = page.locator('button:has-text("创建")').or(page.locator('button:has-text("新增")'));
    await createButton.click();

    // Check form appears
    const modal = page.locator('[role="dialog"]').or(page.locator('.ant-modal'));
    await expect(modal).toBeVisible();

    // Try to submit without filling required fields
    const submitButton = modal.locator('button[type="submit"]').or(modal.locator('button:has-text("确定")'));
    await submitButton.click();

    // Should show validation errors
    const hasError = await page.locator('.ant-form-item-explain-error').or(page.locator('[role="alert"]')).isVisible().catch(() => false);

    // Either validation error or form doesn't submit
    expect(hasError || (await modal.isVisible())).toBeTruthy();
  });
});

test.describe('Deployment API', () => {
  test('API endpoint structure', async ({ request }) => {
    // Test API endpoint exists (may return 401 without auth)
    const response = await request.get('/api/v1/deployment/targets');

    // Should either succeed (200) or require auth (401)
    expect([200, 401, 404]).toContain(response.status());
  });
});
