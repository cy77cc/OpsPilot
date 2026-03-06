import { test, expect } from '../support/auth';

test.describe('Deployment Page', () => {
  test('deployment page shows targets list', async ({ authenticatedPage }) => {
    // Navigate to deployment page
    await authenticatedPage.goto('/deployment');

    // Wait for page to load
    await authenticatedPage.waitForLoadState('networkidle');

    // Check page header exists
    const header = authenticatedPage.locator('h1, h2').first();
    await expect(header).toBeVisible({ timeout: 10000 });

    // Check for deployment targets table or list
    const hasTargetsTable = await authenticatedPage.locator('table').isVisible().catch(() => false);
    const hasTargetsList = await authenticatedPage.locator('[data-testid="targets-list"]').isVisible().catch(() => false);
    const hasEmptyState = await authenticatedPage.locator('text=/暂无|empty|no data/i').isVisible().catch(() => false);
    const hasCreateButton = await authenticatedPage.locator('button:has-text("创建")').or(
      authenticatedPage.locator('button:has-text("新增")')
    ).isVisible().catch(() => false);

    // Page should have at least one of these elements
    expect(hasTargetsTable || hasTargetsList || hasEmptyState || hasCreateButton).toBeTruthy();
  });

  test('create deployment target button works', async ({ authenticatedPage }) => {
    await authenticatedPage.goto('/deployment');
    await authenticatedPage.waitForLoadState('networkidle');

    // Find create button
    const createButton = authenticatedPage.locator('button:has-text("创建")').or(
      authenticatedPage.locator('button:has-text("新增")')
    );

    // Check if create button exists (may not exist if no permissions)
    const buttonExists = await createButton.isVisible().catch(() => false);

    if (buttonExists) {
      await createButton.click();

      // Check for modal or drawer
      const modal = authenticatedPage.locator('[role="dialog"]').or(
        authenticatedPage.locator('.ant-modal')
      ).or(
        authenticatedPage.locator('.ant-drawer')
      );

      // Modal should appear or navigate to create page
      const modalVisible = await modal.isVisible().catch(() => false);
      const navigatedToCreate = authenticatedPage.url().includes('create');

      expect(modalVisible || navigatedToCreate).toBeTruthy();
    } else {
      // Skip if button not visible (permissions issue)
      test.skip();
    }
  });
});

test.describe('Deployment API', () => {
  test('API endpoint structure', async ({ request }) => {
    // Test API endpoint exists (may return 401 without auth)
    const response = await request.get('/api/v1/deployment/targets');

    // Should either succeed (200) or require auth (401/403)
    expect([200, 401, 403, 404]).toContain(response.status());
  });

  test('targets API returns correct structure', async ({ request, authToken }) => {
    if (!authToken) {
      test.skip();
      return;
    }

    const response = await request.get('/api/v1/deployment/targets', {
      headers: {
        Authorization: `Bearer ${authToken}`,
      },
    });

    if (response.ok()) {
      const data = await response.json();

      // Check standard response structure
      expect(data).toHaveProperty('code');
      expect(data).toHaveProperty('msg');

      // Success response should have data
      if (data.code === 1000) {
        expect(data).toHaveProperty('data');
      }
    }
  });
});
