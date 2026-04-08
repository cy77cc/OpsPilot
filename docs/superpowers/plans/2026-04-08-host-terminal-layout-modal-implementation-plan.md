# Host Terminal Layout + File Modal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make HostTerminalPage terminal-first (70/30), remove bottom clipping risk with viewport-height layout, and move file editing from inline panel to a large Modal.

**Architecture:** Keep existing HostTerminalPage API/data flow, but split UI responsibilities: right panel only does file list operations; file content editing is managed in modal-specific state. Preserve all existing host file operations and terminal websocket behavior while tightening container height/min-height chains.

**Tech Stack:** React 19, Ant Design 6, Monaco Editor, Vitest + Testing Library, TypeScript.

---

## File Structure (lock before coding)

- **Modify:** `web/src/pages/Hosts/HostTerminalPage.tsx`
  - Responsibility: adjust layout hierarchy (`100vh` + `minHeight:0` chain), remove inline editor card, add file editor modal state/render/save/close-confirm logic.
- **Create:** `web/src/pages/Hosts/HostTerminalPage.test.tsx`
  - Responsibility: regression tests for modal editing flow and key layout style contract.

No new dependency or API surface change.

---

### Task 1: Add failing tests for modal edit flow and viewport layout contract

**Files:**
- Create: `web/src/pages/Hosts/HostTerminalPage.test.tsx`
- Test: `web/src/pages/Hosts/HostTerminalPage.test.tsx`

- [ ] **Step 1: Write failing test scaffold with stable mocks (API + Monaco + xterm + websocket)**

```tsx
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import HostTerminalPage from './HostTerminalPage';

const mockApi = vi.hoisted(() => ({
  hosts: {
    getHostDetail: vi.fn(),
    createTerminalSession: vi.fn(),
    closeTerminalSession: vi.fn(),
    listFiles: vi.fn(),
    readFile: vi.fn(),
    writeFile: vi.fn(),
    deletePath: vi.fn(),
    renamePath: vi.fn(),
    uploadFile: vi.fn(),
    mkdir: vi.fn(),
    downloadFile: vi.fn(),
  },
}));

vi.mock('../../api', () => ({ Api: mockApi }));
vi.mock('@monaco-editor/react', () => ({
  default: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
    <textarea aria-label="modal-editor" value={value} onChange={(e) => onChange(e.target.value)} />
  ),
}));

vi.mock('xterm', () => ({
  Terminal: class {
    cols = 120;
    rows = 40;
    loadAddon() {}
    open() {}
    focus() {}
    writeln() {}
    write() {}
    onData() { return { dispose() {} }; }
    dispose() {}
  },
}));

vi.mock('xterm-addon-fit', () => ({
  FitAddon: class { fit() {} },
}));

class WebSocketMock {
  static OPEN = 1;
  readyState = 1;
  onopen: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  onclose: (() => void) | null = null;
  send() {}
  close() { this.onclose?.(); }
}
(globalThis as unknown as { WebSocket: unknown }).WebSocket = WebSocketMock as unknown as typeof WebSocket;
```

- [ ] **Step 2: Add failing test that verifies clicking file opens modal editor and save calls write API**

```tsx
it('opens modal editor after clicking a file and saves content', async () => {
  mockApi.hosts.getHostDetail.mockResolvedValue({ data: { id: '1', name: 'node-1', ip: '10.0.0.1' } });
  mockApi.hosts.createTerminalSession.mockResolvedValue({ data: { session_id: 's1', ws_path: '/ws/host' } });
  mockApi.hosts.listFiles.mockResolvedValue({
    data: {
      path: '.',
      list: [{ name: 'app.yaml', path: 'app.yaml', is_dir: false, size: 10, mode: '-rw-r--r--' }],
    },
  });
  mockApi.hosts.readFile.mockResolvedValue({ data: { content: 'kind: ConfigMap' } });
  mockApi.hosts.writeFile.mockResolvedValue({ data: {} });

  render(
    <MemoryRouter initialEntries={['/deployment/infrastructure/hosts/1/terminal']}>
      <Routes>
        <Route path="/deployment/infrastructure/hosts/:id/terminal" element={<HostTerminalPage />} />
      </Routes>
    </MemoryRouter>
  );

  await waitFor(() => expect(screen.getByText('app.yaml')).toBeInTheDocument());
  fireEvent.click(screen.getByText('app.yaml'));

  await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
  fireEvent.change(screen.getByLabelText('modal-editor'), { target: { value: 'kind: Secret' } });
  fireEvent.click(screen.getByRole('button', { name: '保存' }));

  await waitFor(() => {
    expect(mockApi.hosts.writeFile).toHaveBeenCalledWith('1', 'app.yaml', 'kind: Secret');
  });
});
```

- [ ] **Step 3: Add failing test for page root layout contract (`height: 100vh`)**

```tsx
it('uses full viewport layout so terminal bottom line is not clipped by page chrome', async () => {
  mockApi.hosts.getHostDetail.mockResolvedValue({ data: { id: '1', name: 'node-1', ip: '10.0.0.1' } });
  mockApi.hosts.createTerminalSession.mockResolvedValue({ data: { session_id: 's1', ws_path: '/ws/host' } });
  mockApi.hosts.listFiles.mockResolvedValue({ data: { path: '.', list: [] } });

  const { container } = render(
    <MemoryRouter initialEntries={['/deployment/infrastructure/hosts/1/terminal']}>
      <Routes>
        <Route path="/deployment/infrastructure/hosts/:id/terminal" element={<HostTerminalPage />} />
      </Routes>
    </MemoryRouter>
  );

  await waitFor(() => expect(screen.getByText('终端与文件')).toBeInTheDocument());
  const root = container.querySelector('.host-terminal-page') as HTMLDivElement;
  expect(root.style.height).toBe('100vh');
});
```

- [ ] **Step 4: Run tests to verify failure before implementation**

Run: `cd web && npm run test:run -- src/pages/Hosts/HostTerminalPage.test.tsx`
Expected: FAIL (missing modal-based editor behavior and/or layout assertion mismatch).

- [ ] **Step 5: Commit failing tests**

```bash
git add web/src/pages/Hosts/HostTerminalPage.test.tsx
git commit -m "Protect host terminal redesign with failing modal/layout tests"
```

---

### Task 2: Implement terminal-first layout and modal editor migration

**Files:**
- Modify: `web/src/pages/Hosts/HostTerminalPage.tsx`
- Test: `web/src/pages/Hosts/HostTerminalPage.test.tsx`

- [ ] **Step 1: Replace inline editor state with modal-specific editor state**

```tsx
// replace existing editor state
const [activeFilePath, setActiveFilePath] = React.useState('');
const [fileModalOpen, setFileModalOpen] = React.useState(false);
const [modalContent, setModalContent] = React.useState('');
const [modalDirty, setModalDirty] = React.useState(false);
const [modalSaving, setModalSaving] = React.useState(false);
```

- [ ] **Step 2: Update `openFile` and `saveFile` to modal flow without API changes**

```tsx
const openFile = async (item: HostFileItem) => {
  if (!id) return;
  if (item.is_dir) {
    await refreshFiles(item.path);
    return;
  }
  try {
    const res = await Api.hosts.readFile(id, item.path);
    setActiveFilePath(item.path);
    setModalContent(res.data.content || '');
    setModalDirty(false);
    setFileModalOpen(true);
  } catch (err) {
    message.error(err instanceof Error ? err.message : '读取文件失败');
  }
};

const saveFile = async () => {
  if (!id || !activeFilePath) return;
  setModalSaving(true);
  try {
    await Api.hosts.writeFile(id, activeFilePath, modalContent);
    setModalDirty(false);
    message.success('文件已保存');
    await refreshFiles(cwd);
  } catch (err) {
    message.error(err instanceof Error ? err.message : '保存失败');
  } finally {
    setModalSaving(false);
  }
};
```

- [ ] **Step 3: Apply full-viewport layout and terminal-first 70/30 split**

```tsx
// remove calc(100vh - 112px)
// const pageHeight = 'calc(100vh - 112px)';

return (
  <div className="fade-in host-terminal-page" style={{ height: '100vh', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
    ...
    <Card style={{ marginBottom: 8, borderRadius: 10, flex: 1, minHeight: 0, overflow: 'hidden' }} ...>
      <Row gutter={12} style={{ height: '100%', minHeight: 0 }} align="stretch">
        <Col xs={24} xl={17} style={{ display: 'flex', minHeight: 0, minWidth: 0 }}>
          {/* terminal */}
        </Col>
        <Col xs={24} xl={7} style={{ display: 'flex', minHeight: 0, minWidth: 0, overflow: 'hidden', height: '100%' }}>
          {/* file manager only */}
        </Col>
      </Row>
    </Card>
  </div>
);
```

- [ ] **Step 4: Remove inline editor card and render modal editor (80vw x 80vh)**

```tsx
<Modal
  open={fileModalOpen}
  title={activeFilePath ? <Text ellipsis={{ tooltip: activeFilePath }}>{`编辑: ${activeFilePath}`}</Text> : '文件编辑'}
  onCancel={() => {
    if (modalDirty) {
      Modal.confirm({
        title: '放弃未保存修改？',
        content: '当前文件尚未保存，确认关闭编辑窗口吗？',
        okText: '放弃修改',
        cancelText: '继续编辑',
        onOk: () => setFileModalOpen(false),
      });
      return;
    }
    setFileModalOpen(false);
  }}
  width="80vw"
  styles={{ body: { height: '80vh', display: 'flex', flexDirection: 'column', minHeight: 0 } }}
  footer={(
    <Space>
      <Button onClick={() => setFileModalOpen(false)}>取消</Button>
      <Button type="primary" icon={<SaveOutlined />} loading={modalSaving} onClick={() => void saveFile()}>保存</Button>
    </Space>
  )}
  destroyOnClose={false}
>
  <div style={{ flex: 1, minHeight: 0 }}>
    <Editor
      height="100%"
      defaultLanguage="yaml"
      value={modalContent}
      onChange={(v) => {
        setModalContent(v || '');
        setModalDirty(true);
      }}
      theme="vs-dark"
      options={{ minimap: { enabled: false }, fontSize: 13 }}
    />
  </div>
  {modalDirty ? <Alert style={{ marginTop: 8 }} type="warning" showIcon message="内容已修改，记得保存。" /> : null}
</Modal>
```

- [ ] **Step 5: Run targeted test to verify pass**

Run: `cd web && npm run test:run -- src/pages/Hosts/HostTerminalPage.test.tsx`
Expected: PASS with both modal flow and height contract tests green.

- [ ] **Step 6: Commit implementation**

```bash
git add web/src/pages/Hosts/HostTerminalPage.tsx
git commit -m "Prioritize host terminal space and move file editing to modal"
```

---

### Task 3: Stabilize close-confirm path + run project checks

**Files:**
- Modify: `web/src/pages/Hosts/HostTerminalPage.tsx`
- Modify: `web/src/pages/Hosts/HostTerminalPage.test.tsx`

- [ ] **Step 1: Add explicit close-confirm helper to keep all close paths consistent**

```tsx
const requestCloseFileModal = React.useCallback(() => {
  if (!modalDirty) {
    setFileModalOpen(false);
    return;
  }
  Modal.confirm({
    title: '放弃未保存修改？',
    content: '当前文件尚未保存，确认关闭编辑窗口吗？',
    okText: '放弃修改',
    cancelText: '继续编辑',
    onOk: () => setFileModalOpen(false),
  });
}, [modalDirty]);
```

Use it in both `onCancel` and footer `取消` button.

- [ ] **Step 2: Add/adjust test to assert dirty-close confirmation opens**

```tsx
it('shows confirm dialog when closing modal with unsaved edits', async () => {
  // setup same as modal open test
  // open file -> change textarea -> click cancel
  fireEvent.click(screen.getByRole('button', { name: '取消' }));
  await waitFor(() => {
    expect(screen.getByText('放弃未保存修改？')).toBeInTheDocument();
  });
});
```

- [ ] **Step 3: Run lint + focused tests + typecheck for touched scope**

Run: `cd web && npm run lint -- src/pages/Hosts/HostTerminalPage.tsx src/pages/Hosts/HostTerminalPage.test.tsx`
Expected: no lint errors.

Run: `cd web && npm run test:run -- src/pages/Hosts/HostTerminalPage.test.tsx src/pages/Hosts/HostDetailPage.test.tsx`
Expected: PASS.

Run: `cd web && npm run build`
Expected: TypeScript build and Vite build succeed.

- [ ] **Step 4: Commit verification hardening**

```bash
git add web/src/pages/Hosts/HostTerminalPage.tsx web/src/pages/Hosts/HostTerminalPage.test.tsx
git commit -m "Guard host file modal close behavior and verify host page regression suite"
```

---

## Spec Coverage Check (self-review)

- **Terminal container too small due to whitespace** → covered by Task 2 Step 3 (full viewport + terminal-first ratio).
- **Bottom content hidden** → covered by Task 2 Step 3 (height/min-height chain contract) and Task 1 layout assertion.
- **File preview redesign to Modal** → covered by Task 2 Step 4 + Task 1 modal open/save test.
- **Keep file ops working** → preserved in Task 2 Step 2 and regression tests in Task 3.

No uncovered requirement from `docs/superpowers/specs/2026-04-08-host-terminal-layout-modal-design.md`.
