import { test, expect } from '@playwright/test';

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

const historyMessages = Array.from({ length: 18 }, (_, index) => ({
  id: `msg-${index}`,
  role: index % 2 === 0 ? 'user' : 'assistant',
  content: `${index % 2 === 0 ? '问题' : '回答'} ${index}\n${'更多内容 '.repeat(12)}`,
  status: 'done',
}));

function sseBody(): string {
  return [
    'event: delta',
    'data: {"content":"这是新回复的第一段。"}',
    '',
    'event: delta',
    'data: {"content":"这是新回复的第二段。"}',
    '',
    'event: done',
    'data: {"content":"完成"}',
    '',
  ].join('\n');
}

test('sending from detached scroll position snaps AI copilot back to bottom', async ({ page }) => {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 1000,
        msg: 'ok',
        data: fakeUser,
      }),
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
        data: {
          token: fakeToken,
          refreshToken: fakeRefreshToken,
        },
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

  await page.route('**/api/v1/ai/sessions', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 1000,
        msg: 'ok',
        data: [{ id: 'sess-1', title: 'Session 1', scene: 'ai' }],
      }),
    });
  });

  await page.route('**/api/v1/ai/sessions?**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 1000,
        msg: 'ok',
        data: [{ id: 'sess-1', title: 'Session 1', scene: 'ai' }],
      }),
    });
  });

  await page.route('**/api/v1/ai/scene-prompts**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 1000, msg: 'ok', data: { prompts: [] } }),
    });
  });

  await page.route('**/api/v1/ai/sessions/sess-1', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 1000, msg: 'ok', data: { messages: historyMessages } }),
    });
  });

  await page.route('**/api/v1/ai/chat', async (route) => {
    await route.fulfill({
      status: 200,
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
      },
      body: sseBody(),
    });
  });

  await page.goto('/login');
  await page.evaluate(([token, refreshToken, user, permissions]) => {
    localStorage.setItem('token', token);
    localStorage.setItem('refreshToken', refreshToken);
    localStorage.setItem('user', JSON.stringify(user));
    localStorage.setItem('permissions', JSON.stringify(permissions));
  }, [fakeToken, fakeRefreshToken, fakeUser, fakePermissions]);

  await page.goto('/help');
  await page.getByRole('button', { name: /AI Assistant/i }).click();

  const scrollContainer = page.getByTestId('copilot-scroll-container');
  await expect(scrollContainer).toBeVisible();
  await page.getByRole('button', { name: '查看历史会话' }).click();
  await page.getByText('Session 1').click();
  await expect(page.getByText('问题 0')).toBeVisible();
  await expect.poll(async () => {
    return scrollContainer.evaluate((el) => el.scrollHeight > el.clientHeight);
  }).toBe(true);

  await scrollContainer.evaluate((el) => {
    el.scrollTop = 0;
    el.dispatchEvent(new Event('scroll'));
  });

  const sender = page.getByPlaceholder('提问或输入 / 使用技能');
  await sender.fill('帮我继续分析');
  await sender.press('Enter');

  await expect(page.getByText('这是新回复的第一段。')).toBeVisible();

  const distanceToBottom = await scrollContainer.evaluate((el) => {
    return el.scrollHeight - el.scrollTop - el.clientHeight;
  });

  expect(distanceToBottom).toBeLessThan(48);
});
