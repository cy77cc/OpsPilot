import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
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
  default: ({ value, onChange }: { value: string; onChange: (v: string | undefined) => void }) => (
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
    onData() {
      return { dispose() {} };
    }
    dispose() {}
  },
}));

vi.mock('xterm-addon-fit', () => ({
  FitAddon: class { fit() {} },
}));

class WebSocketMock {
  static OPEN = 1;
  static urls: string[] = [];
  readyState = WebSocketMock.OPEN;
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  onclose: (() => void) | null = null;
  constructor(url: string) {
    WebSocketMock.urls.push(url);
  }
  send() {}
  close() {
    this.onclose?.();
  }
}

(globalThis as unknown as { WebSocket: unknown }).WebSocket = WebSocketMock as unknown as typeof WebSocket;

describe('HostTerminalPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    WebSocketMock.urls = [];

    mockApi.hosts.getHostDetail.mockResolvedValue({
      data: { id: '1', name: 'node-1', ip: '10.0.0.1' },
    });
    mockApi.hosts.createTerminalSession.mockResolvedValue({
      data: { session_id: 's1', ws_path: '/ws/host' },
    });
    mockApi.hosts.listFiles.mockResolvedValue({
      data: {
        path: '.',
        list: [
          { name: 'app.yaml', path: 'app.yaml', is_dir: false, size: 10, mode: '-rw-r--r--' },
        ],
      },
    });
    mockApi.hosts.readFile.mockResolvedValue({ data: { content: 'kind: ConfigMap' } });
    mockApi.hosts.writeFile.mockResolvedValue({ data: {} });
  });

  it('opens websocket without appending token query', async () => {
    localStorage.setItem('token', 'sensitive-token');
    mockApi.hosts.createTerminalSession.mockResolvedValue({
      data: { session_id: 's1', ws_path: '/ws/host?trace=1' },
    });

    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/hosts/1/terminal']}>
        <Routes>
          <Route path="/deployment/infrastructure/hosts/:id/terminal" element={<HostTerminalPage />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => expect(WebSocketMock.urls.length).toBeGreaterThan(0));
    expect(WebSocketMock.urls[0]).toContain('/ws/host?trace=1');
    expect(WebSocketMock.urls[0]).not.toContain('token=');
  });

  it('opens modal editor after clicking a file and saves content', async () => {
    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/hosts/1/terminal']}>
        <Routes>
          <Route path="/deployment/infrastructure/hosts/:id/terminal" element={<HostTerminalPage />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => expect(screen.getAllByTitle('app.yaml').length).toBeGreaterThan(0));
    fireEvent.click(screen.getAllByTitle('app.yaml')[0]);

    await waitFor(() => expect(screen.getAllByRole('dialog').length).toBeGreaterThan(0));
    const dialog = screen.getAllByRole('dialog')[0];
    fireEvent.change(screen.getByLabelText('modal-editor'), { target: { value: 'kind: Secret' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /保存/ }));

    await waitFor(() => {
      expect(mockApi.hosts.writeFile).toHaveBeenCalledWith('1', 'app.yaml', 'kind: Secret');
    });
  });

  it('shows host-key trust confirmation when opening a file requires trust', async () => {
    mockApi.hosts.readFile.mockRejectedValueOnce({
      businessCode: 2000,
      message: 'ssh host key verification failed',
      details: {
        error_type: 'ssh_host_key_unknown',
        host_key: {
          host: '10.0.0.1',
          port: 22,
          algorithm: 'ssh-ed25519',
          fingerprint_sha256: 'SHA256:test',
          public_key: 'ssh-ed25519 AAAATEST',
        },
      },
    });

    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/hosts/1/terminal']}>
        <Routes>
          <Route path="/deployment/infrastructure/hosts/:id/terminal" element={<HostTerminalPage />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => expect(screen.getAllByTitle('app.yaml').length).toBeGreaterThan(0));
    fireEvent.click(screen.getAllByTitle('app.yaml')[0]);

    expect(await screen.findByText('信任此主机指纹？')).toBeInTheDocument();
  });

  it('uses full viewport layout so terminal bottom line is not clipped by page chrome', async () => {
    const { container } = render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/hosts/1/terminal']}>
        <Routes>
          <Route path="/deployment/infrastructure/hosts/:id/terminal" element={<HostTerminalPage />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => expect(mockApi.hosts.listFiles).toHaveBeenCalled());
    const root = container.querySelector('.host-terminal-page') as HTMLDivElement;
    expect(root.style.height).toContain('100dvh');
    expect(root.style.height).toContain('--app-shell-offset');
  });

  it('shows confirm dialog when closing modal with unsaved edits', async () => {
    render(
      <MemoryRouter initialEntries={['/deployment/infrastructure/hosts/1/terminal']}>
        <Routes>
          <Route path="/deployment/infrastructure/hosts/:id/terminal" element={<HostTerminalPage />} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() => expect(screen.getAllByTitle('app.yaml').length).toBeGreaterThan(0));
    fireEvent.click(screen.getAllByTitle('app.yaml')[0]);
    await waitFor(() => expect(screen.getByLabelText('modal-editor')).toBeInTheDocument());
    const dialog = screen.getByLabelText('modal-editor').closest('[role="dialog"]');
    expect(dialog).not.toBeNull();

    fireEvent.change(screen.getByLabelText('modal-editor'), { target: { value: 'kind: Secret' } });
    fireEvent.click(within(dialog as HTMLElement).getByRole('button', { name: /取\s*消/ }));

    await waitFor(() => {
      expect(screen.getByLabelText('modal-editor')).toBeInTheDocument();
    });
    expect(mockApi.hosts.writeFile).not.toHaveBeenCalled();
  });
});
