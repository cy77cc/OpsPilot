import { test, expect, Page } from '@playwright/test';

// =============================================================================
// Mock Data & Utilities
// =============================================================================

const fakePermissions = ['*:*'];

const fakeUser = {
  id: 1,
  username: 'admin',
  name: 'Admin',
  nickname: 'Admin',
  email: 'admin@example.com',
  status: 'active',
  roles: ['admin'],
  permissions: fakePermissions,
};

function makeJwt(expSecondsFromNow: number): string {
  const header = Buffer.from(JSON.stringify({ alg: 'HS256', typ: 'JWT' })).toString('base64url');
  const payload = Buffer.from(JSON.stringify({
    sub: '1',
    exp: Math.floor(Date.now() / 1000) + expSecondsFromNow,
  })).toString('base64url');
  return `${header}.${payload}.signature`;
}

const fakeToken = makeJwt(60 * 60);
const fakeRefreshToken = 'playwright-refresh-token';

/**
 * Setup authentication and common mocks for all tests
 */
async function setupCommonMocks(page: Page) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 1000, msg: 'ok', data: fakeUser }),
    });
  });

  await page.route('**/api/v1/rbac/me/permissions', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 1000, msg: 'ok', data: fakePermissions }),
    });
  });

  await page.route('**/api/v1/auth/refresh', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 1000,
        msg: 'ok',
        data: { token: fakeToken, refreshToken: fakeRefreshToken },
      }),
    });
  });

  await page.route('**/api/v1/notifications**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
    });
  });
}

/**
 * Setup initial localStorage with auth tokens
 * This must be called after navigating to a page (like /login)
 */
async function setupAuth(page: Page) {
  await page.evaluate(([token, refreshToken, user, permissions]) => {
    localStorage.setItem('token', token);
    localStorage.setItem('refreshToken', refreshToken);
    localStorage.setItem('user', JSON.stringify(user));
    localStorage.setItem('permissions', JSON.stringify(permissions));
  }, [fakeToken, fakeRefreshToken, fakeUser, fakePermissions]);
}

/**
 * Login helper - navigate to login page, set auth, then navigate to target page
 */
async function loginAndNavigate(page: Page, targetPath: string = '/help') {
  await page.goto('/login');
  await setupAuth(page);
  await page.goto(targetPath);
  // Wait for page to be ready
  await page.waitForLoadState('networkidle');
}

/**
 * Open AI Copilot drawer
 */
async function openAICopilot(page: Page) {
  const button = page.getByRole('button', { name: /AI Assistant/i });
  await button.click();
  await expect(page.getByTestId('copilot-scroll-container')).toBeVisible({ timeout: 10000 });
}

// =============================================================================
// Test Suite: AI Chat Flow
// =============================================================================

test.describe('AI Chat Flow', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page);
    await loginAndNavigate(page, '/help');
  });

  test('should create new session and send message', async ({ page }) => {
    // Mock empty sessions list
    await page.route('**/api/v1/ai/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
      });
    });

    // Mock session creation
    await page.route('**/api/v1/ai/sessions', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 1000,
            msg: 'ok',
            data: { id: 'new-session-123', title: 'Test message', scene: 'ai' },
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
        });
      }
    });

    // Mock chat SSE stream
    await page.route('**/api/v1/ai/chat', async (route) => {
      const sseBody = [
        'event: meta',
        'data: {"session_id":"new-session-123","run_id":"run-456","turn":1}',
        '',
        'event: delta',
        'data: {"content":"Hello! How can I help you?"}',
        '',
        'event: done',
        'data: {}',
        '',
      ].join('\n');

      await route.fulfill({
        status: 200,
        headers: {
          'Content-Type': 'text/event-stream',
          'Cache-Control': 'no-cache',
        },
        body: sseBody,
      });
    });

    await openAICopilot(page);

    // Send a message
    const sender = page.getByPlaceholder('提问或输入 / 使用技能');
    await sender.fill('Test message');
    await sender.press('Enter');

    // Verify response appears
    await expect(page.getByText('Hello! How can I help you?')).toBeVisible({ timeout: 15000 });
  });

  test('should load and display conversation history', async ({ page }) => {
    const historyMessages = [
      { id: 'msg-1', role: 'user', content: 'Previous question', status: 'done' },
      { id: 'msg-2', role: 'assistant', content: 'Previous answer', status: 'done' },
    ];

    await page.route('**/api/v1/ai/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 1000,
          msg: 'ok',
          data: [{ id: 'session-1', title: 'Previous Session', scene: 'ai' }],
        }),
      });
    });

    await page.route('**/api/v1/ai/sessions/session-1', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 1000, msg: 'ok', data: { messages: historyMessages } }),
      });
    });

    await openAICopilot(page);

    // Open history popover
    await page.getByRole('button', { name: '查看历史会话' }).click();
    await page.getByText('Previous Session').click();

    // Verify history loaded
    await expect(page.getByText('Previous question')).toBeVisible();
    await expect(page.getByText('Previous answer')).toBeVisible();
  });

  test('should handle streaming response with delta events', async ({ page }) => {
    await page.route('**/api/v1/ai/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
      });
    });

    await page.route('**/api/v1/ai/sessions', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 1000,
            msg: 'ok',
            data: { id: 'stream-session', title: 'Stream test', scene: 'ai' },
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
        });
      }
    });

    // Mock streaming SSE
    await page.route('**/api/v1/ai/chat', async (route) => {
      const sseBody = [
        'event: meta',
        'data: {"session_id":"stream-session","run_id":"run-1","turn":1}',
        '',
        'event: delta',
        'data: {"content":"First chunk "}',
        '',
        'event: delta',
        'data: {"content":"second chunk "}',
        '',
        'event: delta',
        'data: {"content":"third chunk."}',
        '',
        'event: done',
        'data: {}',
        '',
      ].join('\n');

      await route.fulfill({
        status: 200,
        headers: {
          'Content-Type': 'text/event-stream',
          'Cache-Control': 'no-cache',
        },
        body: sseBody,
      });
    });

    await openAICopilot(page);

    const sender = page.getByPlaceholder('提问或输入 / 使用技能');
    await sender.fill('Stream test');
    await sender.press('Enter');

    // Verify chunks appear (combined text)
    await expect(page.getByText(/First chunk.*second chunk.*third chunk/)).toBeVisible({ timeout: 15000 });
  });
});

// =============================================================================
// Test Suite: Approval Flow (Human-in-the-Loop)
// NOTE: These tests require real SSE streaming which Playwright's route.fulfill
// doesn't support well. They are marked as fixme until we can use a proper SSE mock.
// =============================================================================

test.describe('Approval Flow', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page);
    await loginAndNavigate(page, '/help');
  });

  test.fixme('should show approval UI when tool_approval event received', async ({ page }) => {
    await page.route('**/api/v1/ai/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
      });
    });

    await page.route('**/api/v1/ai/sessions', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 1000,
            msg: 'ok',
            data: { id: 'approval-session', title: 'Approval test', scene: 'ai' },
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
        });
      }
    });

    // Mock SSE with tool_call and tool_approval events
    // Use correct event format matching A2UIToolApprovalEvent interface
    await page.route('**/api/v1/ai/chat', async (route) => {
      const sseBody = [
        'event: meta',
        'data: {"session_id":"approval-session","run_id":"run-approval","turn":1}',
        '',
        'event: tool_call',
        'data: {"call_id":"call-123","tool_name":"k8s_scale_deployment","arguments":{"cluster_id":1,"namespace":"default","name":"nginx","replicas":3}}',
        '',
        'event: tool_approval',
        'data: {"call_id":"call-123","tool_name":"k8s_scale_deployment","approval_id":"approval-456","preview":{"cluster_id":1,"namespace":"default","name":"nginx","replicas":3},"timeout_seconds":300}',
        '',
      ].join('\n');

      await route.fulfill({
        status: 200,
        headers: {
          'Content-Type': 'text/event-stream',
          'Cache-Control': 'no-cache',
        },
        body: sseBody,
      });
    });

    await openAICopilot(page);

    const sender = page.getByPlaceholder('提问或输入 / 使用技能');
    await sender.fill('Scale nginx deployment');
    await sender.press('Enter');

    // Wait for tool reference to appear
    await expect(page.getByText('k8s_scale_deployment')).toBeVisible({ timeout: 15000 });

    // Verify approval buttons appear (with "待审批" status)
    await expect(page.getByRole('button', { name: '批准' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: '拒绝' })).toBeVisible();
  });

  test.fixme('should approve and show success state', async ({ page }) => {
    await page.route('**/api/v1/ai/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
      });
    });

    await page.route('**/api/v1/ai/sessions', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 1000,
            msg: 'ok',
            data: { id: 'approve-session', title: 'Approve test', scene: 'ai' },
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
        });
      }
    });

    // Track approval submission
    let approvalSubmitted = false;
    await page.route('**/api/v1/ai/approvals/approval-456/submit', async (route) => {
      approvalSubmitted = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 1000,
          msg: 'ok',
          data: { status: 'approved' },
        }),
      });
    });

    // Mock SSE with approval flow
    await page.route('**/api/v1/ai/chat', async (route) => {
      const sseBody = [
        'event: meta',
        'data: {"session_id":"approve-session","run_id":"run-approve","turn":1}',
        '',
        'event: tool_call',
        'data: {"call_id":"call-approve","tool_name":"k8s_scale_deployment","arguments":{}}',
        '',
        'event: tool_approval',
        'data: {"call_id":"call-approve","tool_name":"k8s_scale_deployment","approval_id":"approval-456","preview":{},"timeout_seconds":300}',
        '',
      ].join('\n');

      await route.fulfill({
        status: 200,
        headers: {
          'Content-Type': 'text/event-stream',
          'Cache-Control': 'no-cache',
        },
        body: sseBody,
      });
    });

    await openAICopilot(page);

    const sender = page.getByPlaceholder('提问或输入 / 使用技能');
    await sender.fill('Scale deployment');
    await sender.press('Enter');

    // Wait for approval UI
    await expect(page.getByRole('button', { name: '批准' })).toBeVisible({ timeout: 15000 });

    // Click approve
    await page.getByRole('button', { name: '批准' }).click();

    // Verify approval was submitted
    await expect.poll(() => approvalSubmitted, { timeout: 10000 }).toBe(true);

    // Verify approved state shows
    await expect(page.getByText('已批准')).toBeVisible({ timeout: 10000 });
  });

  test.fixme('should reject and show rejected state', async ({ page }) => {
    await page.route('**/api/v1/ai/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
      });
    });

    await page.route('**/api/v1/ai/sessions', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 1000,
            msg: 'ok',
            data: { id: 'reject-session', title: 'Reject test', scene: 'ai' },
          }),
        });
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
        });
      }
    });

    // Track rejection submission
    let rejectionSubmitted = false;
    await page.route('**/api/v1/ai/approvals/approval-reject/submit', async (route) => {
      rejectionSubmitted = true;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 1000,
          msg: 'ok',
          data: { status: 'rejected' },
        }),
      });
    });

    // Mock SSE with approval flow
    await page.route('**/api/v1/ai/chat', async (route) => {
      const sseBody = [
        'event: meta',
        'data: {"session_id":"reject-session","run_id":"run-reject","turn":1}',
        '',
        'event: tool_call',
        'data: {"call_id":"call-reject","tool_name":"k8s_delete_pod","arguments":{}}',
        '',
        'event: tool_approval',
        'data: {"call_id":"call-reject","tool_name":"k8s_delete_pod","approval_id":"approval-reject","preview":{},"timeout_seconds":300}',
        '',
      ].join('\n');

      await route.fulfill({
        status: 200,
        headers: {
          'Content-Type': 'text/event-stream',
          'Cache-Control': 'no-cache',
        },
        body: sseBody,
      });
    });

    await openAICopilot(page);

    const sender = page.getByPlaceholder('提问或输入 / 使用技能');
    await sender.fill('Delete pod');
    await sender.press('Enter');

    // Wait for approval UI
    await expect(page.getByRole('button', { name: '拒绝' })).toBeVisible({ timeout: 15000 });

    // Click reject
    await page.getByRole('button', { name: '拒绝' }).click();

    // Verify rejection was submitted
    await expect.poll(() => rejectionSubmitted, { timeout: 10000 }).toBe(true);

    // Verify rejected state shows
    await expect(page.getByText('已拒绝')).toBeVisible({ timeout: 10000 });
  });
});

// =============================================================================
// Test Suite: Session Management
// =============================================================================

test.describe('Session Management', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page);
    await loginAndNavigate(page, '/help');
  });

  test('should create new session when clicking new button', async ({ page }) => {
    await page.route('**/api/v1/ai/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 1000,
          msg: 'ok',
          data: [{ id: 'existing-session', title: 'Existing Session', scene: 'ai' }],
        }),
      });
    });

    await openAICopilot(page);

    // Click new session button
    await page.getByRole('button', { name: '新建会话' }).click();

    // Verify we're on a new session (welcome message visible)
    await expect(page.getByText('你好，我是您的智能运维助手!')).toBeVisible();
  });

  test('should switch between sessions', async ({ page }) => {
    await page.route('**/api/v1/ai/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 1000,
          msg: 'ok',
          data: [
            { id: 'session-1', title: 'First Session', scene: 'ai' },
            { id: 'session-2', title: 'Second Session', scene: 'ai' },
          ],
        }),
      });
    });

    await page.route('**/api/v1/ai/sessions/session-1', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 1000,
          msg: 'ok',
          data: {
            id: 'session-1',
            title: 'First Session',
            messages: [{ id: 'm1', role: 'user', content: 'First session message', status: 'done' }],
          },
        }),
      });
    });

    await page.route('**/api/v1/ai/sessions/session-2', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 1000,
          msg: 'ok',
          data: {
            id: 'session-2',
            title: 'Second Session',
            messages: [{ id: 'm2', role: 'user', content: 'Second session message', status: 'done' }],
          },
        }),
      });
    });

    await openAICopilot(page);

    // Open history popover
    await page.getByRole('button', { name: '查看历史会话' }).click();

    // Click second session
    await page.getByText('Second Session').click();

    // Verify second session loaded
    await expect(page.getByText('Second session message')).toBeVisible();
  });
});

// =============================================================================
// Test Suite: Scene Context
// =============================================================================

test.describe('Scene Context', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page);
  });

  test('should detect host scene on hosts page', async ({ page }) => {
    await page.route('**/api/v1/ai/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
      });
    });

    await loginAndNavigate(page, '/hosts');
    await openAICopilot(page);

    // Verify scene tag shows host
    await expect(page.getByText('host')).toBeVisible();
  });

  // FIXME: Scene detection for /k8s path needs investigation
  test.skip('should detect cluster scene on clusters page', async ({ page }) => {
    await page.route('**/api/v1/ai/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
      });
    });

    await loginAndNavigate(page, '/k8s');
    await openAICopilot(page);

    // Verify scene tag shows cluster
    await expect(page.getByText('cluster')).toBeVisible();
  });
});

// =============================================================================
// Test Suite: UI Interactions
// =============================================================================

test.describe('UI Interactions', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page);
    await loginAndNavigate(page, '/help');
  });

  test('should close drawer when clicking close button', async ({ page }) => {
    await page.route('**/api/v1/ai/sessions**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 1000, msg: 'ok', data: [] }),
      });
    });

    await openAICopilot(page);

    // Close drawer by clicking the X button in the header
    const closeButton = page.locator('.ant-drawer-header').getByRole('button').first();
    await closeButton.click();

    // Verify drawer is closed
    await expect(page.getByTestId('copilot-scroll-container')).not.toBeVisible();
  });
});
