import React from 'react';
import { Alert, Breadcrumb, Button, Card, Col, Input, Modal, Popconfirm, Row, Space, Spin, Tag, Typography, Upload, message } from 'antd';
import { ArrowLeftOutlined, DeleteOutlined, DownloadOutlined, EditOutlined, FileAddOutlined, FolderAddOutlined, ReloadOutlined, SaveOutlined, UploadOutlined } from '@ant-design/icons';
import { Link, useNavigate, useParams } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import { Api } from '../../api';
import type { Host, HostFileItem } from '../../api/modules/hosts';
import HostKeyTrustModal from '../../components/Hosts/HostKeyTrustModal';
import { useStableFetch } from '../../hooks';
import { parseHostKeyTrustError, useHostKeyTrust } from '../../hooks/useHostKeyTrust';

const { Text } = Typography;
const TERMINAL_INPUT_BATCH_MS = 16;

type ConnStatus = 'idle' | 'connecting' | 'connected' | 'closed' | 'error';

const HostTerminalPage: React.FC = () => {
  const navigate = useNavigate();
  const { id = '' } = useParams<{ id: string }>();
  const xtermRef = React.useRef<Terminal | null>(null);
  const fitRef = React.useRef<FitAddon | null>(null);
  const resizeObserverRef = React.useRef<ResizeObserver | null>(null);
  const inputListenerRef = React.useRef<{ dispose: () => void } | null>(null);
  const wsRef = React.useRef<WebSocket | null>(null);
  const inputBufferRef = React.useRef('');
  const inputFlushTimerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);
  const terminalInitTimerRef = React.useRef<ReturnType<typeof setTimeout> | null>(null);
  const termWrapRef = React.useRef<HTMLDivElement>(null);
  const filePaneRef = React.useRef<HTMLDivElement>(null);
  const unmountedRef = React.useRef(false);
  const [status, setStatus] = React.useState<ConnStatus>('idle');
  const [host, setHost] = React.useState<Host | null>(null);
  const [sessionID, setSessionID] = React.useState('');
  const [cwd, setCwd] = React.useState('.');
  const [files, setFiles] = React.useState<HostFileItem[]>([]);
  const [activeFilePath, setActiveFilePath] = React.useState('');
  const [fileModalOpen, setFileModalOpen] = React.useState(false);
  const [modalContent, setModalContent] = React.useState('');
  const [filesLoading, setFilesLoading] = React.useState(false);
  const [modalDirty, setModalDirty] = React.useState(false);
  const [modalSaving, setModalSaving] = React.useState(false);
  const [newDirOpen, setNewDirOpen] = React.useState(false);
  const [newDirName, setNewDirName] = React.useState('');
  const [pathInput, setPathInput] = React.useState('.');
  const [filePaneWidth, setFilePaneWidth] = React.useState(0);
  const retryOperationRef = React.useRef<() => Promise<void>>(async () => {});
  const {
    pendingTrust,
    setPendingTrust,
    confirming,
    runWithTrustRetry,
    confirmTrustAndRetry,
  } = useHostKeyTrust(id);

  const fileColumnMode = React.useMemo<'full' | 'compact' | 'minimal'>(() => {
    if (filePaneWidth > 0 && filePaneWidth < 420) return 'minimal';
    if (filePaneWidth > 0 && filePaneWidth < 560) return 'compact';
    return 'full';
  }, [filePaneWidth]);
  const showMTime = fileColumnMode !== 'minimal';
  const showSize = fileColumnMode === 'full';
  const fileGridColumns = fileColumnMode === 'full'
    ? 'minmax(120px, 1fr) 108px 72px 88px'
    : fileColumnMode === 'compact'
      ? 'minmax(140px, 1fr) 88px 88px'
      : 'minmax(140px, 1fr) 88px';

  const setupTerminal = React.useCallback(() => {
    if (!termWrapRef.current || xtermRef.current) return false;
    if (termWrapRef.current.clientWidth === 0 || termWrapRef.current.clientHeight === 0) {
      return false;
    }
    const term = new Terminal({
      cursorBlink: true,
      convertEol: true,
      fontFamily: 'JetBrains Mono, Menlo, Monaco, Consolas, monospace',
      fontSize: 13,
      theme: {
        background: '#0e1117',
        foreground: '#d4d4d4',
        cursor: '#8ae234',
      },
    });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(termWrapRef.current);
    // Use guarded fit to avoid xterm viewport crashes during StrictMode mount/unmount cycles.
    if (termWrapRef.current.clientWidth > 0 && termWrapRef.current.clientHeight > 0) {
      try {
        fitAddon.fit();
      } catch {
        // Ignore transient fit errors when terminal is tearing down.
      }
    }
    term.writeln('\x1b[90mConnecting to host terminal...\x1b[0m');
    xtermRef.current = term;
    fitRef.current = fitAddon;
    return true;
  }, []);

  const safeFit = React.useCallback(() => {
    const term = xtermRef.current;
    const fit = fitRef.current;
    const wrap = termWrapRef.current;
    if (!term || !fit || !wrap || !wrap.isConnected) return;
    // Skip fit when container is detached/collapsed to avoid xterm internal dimension errors.
    if (wrap.clientWidth === 0 || wrap.clientHeight === 0) return;
    try {
      fit.fit();
    } catch {
      // no-op: fit can throw if terminal is disposed during async layout updates
    }
  }, []);

  const clearPendingTerminalInput = React.useCallback(() => {
    if (inputFlushTimerRef.current) {
      clearTimeout(inputFlushTimerRef.current);
      inputFlushTimerRef.current = null;
    }
    inputBufferRef.current = '';
  }, []);

  const flushPendingTerminalInput = React.useCallback((ws?: WebSocket | null) => {
    if (inputFlushTimerRef.current) {
      clearTimeout(inputFlushTimerRef.current);
      inputFlushTimerRef.current = null;
    }
    const buffered = inputBufferRef.current;
    inputBufferRef.current = '';
    if (!buffered) return;
    const target = ws ?? wsRef.current;
    if (!target || target.readyState !== WebSocket.OPEN) return;
    target.send(JSON.stringify({ type: 'input', input: buffered }));
  }, []);

  const queueTerminalInput = React.useCallback((ws: WebSocket, data: string) => {
    if (!data) return;
    inputBufferRef.current += data;
    if (data.includes('\r') || data.includes('\u0003') || data.includes('\u0004')) {
      flushPendingTerminalInput(ws);
      return;
    }
    if (inputFlushTimerRef.current) return;
    inputFlushTimerRef.current = setTimeout(() => {
      flushPendingTerminalInput(ws);
    }, TERMINAL_INPUT_BATCH_MS);
  }, [flushPendingTerminalInput]);

  React.useEffect(() => {
    let cancelled = false;
    unmountedRef.current = false;
    const tryInit = () => {
      if (cancelled) return;
      if (!setupTerminal()) {
        terminalInitTimerRef.current = setTimeout(tryInit, 50);
      } else {
        terminalInitTimerRef.current = null;
      }
    };
    terminalInitTimerRef.current = setTimeout(tryInit, 0);
    const onResize = () => safeFit();
    if (typeof window !== 'undefined') {
      window.addEventListener('resize', onResize);
    }
    return () => {
      cancelled = true;
      unmountedRef.current = true;
      if (terminalInitTimerRef.current) {
        clearTimeout(terminalInitTimerRef.current);
        terminalInitTimerRef.current = null;
      }
      if (typeof window !== 'undefined') {
        window.removeEventListener('resize', onResize);
      }
      clearPendingTerminalInput();
      wsRef.current?.close();
      wsRef.current = null;
      resizeObserverRef.current?.disconnect();
      resizeObserverRef.current = null;
      inputListenerRef.current?.dispose();
      inputListenerRef.current = null;
      xtermRef.current?.dispose();
      xtermRef.current = null;
      fitRef.current = null;
    };
  }, [clearPendingTerminalInput, safeFit, setupTerminal]);

  React.useEffect(() => {
    const pane = filePaneRef.current;
    if (!pane) return;
    const update = () => setFilePaneWidth(pane.clientWidth);
    update();
    if (typeof ResizeObserver === 'undefined') {
      return;
    }
    const observer = new ResizeObserver(update);
    observer.observe(pane);
    return () => observer.disconnect();
  }, []);

  React.useEffect(() => {
    if (status !== 'connected' || typeof window === 'undefined') return;
    const raf = window.requestAnimationFrame(() => safeFit());
    return () => window.cancelAnimationFrame(raf);
  }, [status, filePaneWidth, safeFit]);

  const wsURLFromPath = (wsPath: string) => {
    if (typeof window === 'undefined') {
      return wsPath;
    }
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    return `${protocol}://${window.location.host}${wsPath}`;
  };

  const refreshFiles = React.useCallback(async (dirPath: string) => {
    if (!id || unmountedRef.current) return;
    setFilesLoading(true);
    try {
      retryOperationRef.current = async () => {
        await refreshFiles(dirPath);
      };
      const res = await runWithTrustRetry(() => Api.hosts.listFiles(id, dirPath));
      if (unmountedRef.current) return;
      setFiles(res.data.list || []);
      setCwd(res.data.path || dirPath);
      setPathInput(res.data.path || dirPath);
    } catch (err) {
      if (unmountedRef.current) return;
      if (!parseHostKeyTrustError(err)) {
        message.error(err instanceof Error ? err.message : '加载文件列表失败');
      }
    } finally {
      if (!unmountedRef.current) {
        setFilesLoading(false);
      }
    }
  }, [id, runWithTrustRetry]);

  // Store callbacks in refs to avoid useEffect dependency issues
  const safeFitRef = React.useRef(safeFit);
  const refreshFilesRef = React.useRef(refreshFiles);
  safeFitRef.current = safeFit;
  refreshFilesRef.current = refreshFiles;

  // Use a ref to track connection state across StrictMode remounts
  const connectingRef = React.useRef(false);

  React.useEffect(() => {
    // Prevent duplicate connections
    if (connectingRef.current || !id) {
      return;
    }
    connectingRef.current = true;

    let cancelled = false;

    const doConnect = async () => {
      if (unmountedRef.current) {
        return;
      }
      setStatus('connecting');
      try {
        retryOperationRef.current = async () => {
          await connect();
        };
        const [hostResp, sessResp] = await runWithTrustRetry(() => Promise.all([
          Api.hosts.getHostDetail(id),
          Api.hosts.createTerminalSession(id),
        ]));

        if (cancelled || unmountedRef.current) return;

        setHost(hostResp.data);
        setSessionID(sessResp.data.session_id);

        const ws = new WebSocket(wsURLFromPath(sessResp.data.ws_path));
        wsRef.current = ws;
        ws.onopen = () => {
          if (cancelled || unmountedRef.current) return;
          setStatus('connected');
          safeFitRef.current();
          const term = xtermRef.current;
          if (!term) return;
          term.focus();
          term.writeln(`\x1b[32mConnected to ${hostResp.data.name} (${hostResp.data.ip})\x1b[0m`);
          inputListenerRef.current?.dispose();
          inputListenerRef.current = term.onData((data) => {
            queueTerminalInput(ws, data);
          });
          const fit = fitRef.current;
          const size = term.cols && term.rows ? { cols: term.cols, rows: term.rows } : { cols: 120, rows: 40 };
          ws.send(JSON.stringify({ type: 'resize', ...size }));
          if (fit) {
            resizeObserverRef.current?.disconnect();
            const observer = new ResizeObserver(() => {
              if (xtermRef.current !== term) return;
              safeFitRef.current();
              if (ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
              }
            });
            resizeObserverRef.current = observer;
            if (termWrapRef.current) observer.observe(termWrapRef.current);
          }
        };
        ws.onmessage = (event) => {
          const term = xtermRef.current;
          if (!term) return;
          try {
            const msg = JSON.parse(String(event.data));
            if (msg.type === 'output' && msg.payload?.data) {
              term.write(String(msg.payload.data));
            }
          } catch {
            term.write(String(event.data));
          }
        };
        ws.onerror = () => {
          if (cancelled || unmountedRef.current) return;
          setStatus('error');
          xtermRef.current?.writeln('\r\n\x1b[31mTerminal websocket error\x1b[0m');
        };
        ws.onclose = () => {
          if (cancelled || unmountedRef.current) return;
          setStatus('closed');
          resizeObserverRef.current?.disconnect();
          resizeObserverRef.current = null;
          inputListenerRef.current?.dispose();
          inputListenerRef.current = null;
          clearPendingTerminalInput();
          xtermRef.current?.writeln('\r\n\x1b[90mSession closed\x1b[0m');
          connectingRef.current = false;
        };
        await refreshFilesRef.current('.');
      } catch (err) {
        if (cancelled || unmountedRef.current) return;
        if (parseHostKeyTrustError(err)) {
          connectingRef.current = false;
          return;
        }
        setStatus('error');
        message.error(err instanceof Error ? err.message : '终端连接失败');
        connectingRef.current = false;
      }
    };

    void doConnect();

    return () => {
      cancelled = true;
      // Close websocket on cleanup
      if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
        wsRef.current.close();
      }
      // Reset connecting flag so reconnection can happen
      connectingRef.current = false;
    };
  }, [clearPendingTerminalInput, id, queueTerminalInput, runWithTrustRetry]); // Only depend on id - stable!

  const connect = React.useCallback(async () => {
    // Manual reconnect - reset the flag first
    connectingRef.current = false;
    if (!id || unmountedRef.current) return;

    connectingRef.current = true;
    setStatus('connecting');
    try {
      retryOperationRef.current = async () => {
        await connect();
      };
      const [hostResp, sessResp] = await runWithTrustRetry(() => Promise.all([
        Api.hosts.getHostDetail(id),
        Api.hosts.createTerminalSession(id),
      ]));
      if (unmountedRef.current) return;
      setHost(hostResp.data);
      setSessionID(sessResp.data.session_id);

      const ws = new WebSocket(wsURLFromPath(sessResp.data.ws_path));
      wsRef.current = ws;
      ws.onopen = () => {
        if (unmountedRef.current) return;
        setStatus('connected');
        safeFitRef.current();
        const term = xtermRef.current;
        if (!term) return;
        term.focus();
        term.writeln(`\x1b[32mConnected to ${hostResp.data.name} (${hostResp.data.ip})\x1b[0m`);
        inputListenerRef.current?.dispose();
        inputListenerRef.current = term.onData((data) => {
          queueTerminalInput(ws, data);
        });
        const fit = fitRef.current;
        const size = term.cols && term.rows ? { cols: term.cols, rows: term.rows } : { cols: 120, rows: 40 };
        ws.send(JSON.stringify({ type: 'resize', ...size }));
        if (fit) {
          resizeObserverRef.current?.disconnect();
          const observer = new ResizeObserver(() => {
            if (xtermRef.current !== term) return;
            safeFitRef.current();
            if (ws.readyState === WebSocket.OPEN) {
              ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
            }
          });
          resizeObserverRef.current = observer;
          if (termWrapRef.current) observer.observe(termWrapRef.current);
        }
      };
      ws.onmessage = (event) => {
        const term = xtermRef.current;
        if (!term) return;
        try {
          const msg = JSON.parse(String(event.data));
          if (msg.type === 'output' && msg.payload?.data) {
            term.write(String(msg.payload.data));
          }
        } catch {
          term.write(String(event.data));
        }
      };
      ws.onerror = () => {
        if (unmountedRef.current) return;
        setStatus('error');
        xtermRef.current?.writeln('\r\n\x1b[31mTerminal websocket error\x1b[0m');
      };
      ws.onclose = () => {
        if (unmountedRef.current) return;
        setStatus('closed');
        resizeObserverRef.current?.disconnect();
        resizeObserverRef.current = null;
        inputListenerRef.current?.dispose();
        inputListenerRef.current = null;
        clearPendingTerminalInput();
        xtermRef.current?.writeln('\r\n\x1b[90mSession closed\x1b[0m');
        connectingRef.current = false;
      };
      await refreshFilesRef.current('.');
    } catch (err) {
      if (unmountedRef.current) return;
      if (parseHostKeyTrustError(err)) {
        connectingRef.current = false;
        return;
      }
      setStatus('error');
      message.error(err instanceof Error ? err.message : '终端连接失败');
      connectingRef.current = false;
    }
  }, [clearPendingTerminalInput, id, queueTerminalInput, runWithTrustRetry]);

  React.useEffect(() => {
    if (typeof window === 'undefined') {
      return undefined;
    }
    const raf = window.requestAnimationFrame(() => {
      safeFit();
    });
    return () => window.cancelAnimationFrame(raf);
  }, [safeFit, fileModalOpen]);

  const closeSession = React.useCallback(async () => {
    flushPendingTerminalInput();
    wsRef.current?.close();
    if (id && sessionID) {
      try {
        await Api.hosts.closeTerminalSession(id, sessionID);
      } catch {
        // noop
      }
    }
    if (!unmountedRef.current) {
      setStatus('closed');
    }
  }, [flushPendingTerminalInput, id, sessionID]);

  const openFile = async (item: HostFileItem) => {
    if (!id) return;
    if (item.is_dir) {
      await refreshFiles(item.path);
      return;
    }
    try {
      const operation = async () => {
        const res = await Api.hosts.readFile(id, item.path);
        setActiveFilePath(item.path);
        setModalContent(res.data.content || '');
        setModalDirty(false);
        setFileModalOpen(true);
      };
      retryOperationRef.current = operation;
      await runWithTrustRetry(operation);
    } catch (err) {
      if (!parseHostKeyTrustError(err)) {
        message.error(err instanceof Error ? err.message : '读取文件失败');
      }
    }
  };

  const saveFile = async () => {
    if (!id || !activeFilePath) return;
    setModalSaving(true);
    try {
      const operation = async () => {
        await Api.hosts.writeFile(id, activeFilePath, modalContent);
        setModalDirty(false);
        message.success('文件已保存');
        await refreshFiles(cwd);
      };
      retryOperationRef.current = operation;
      await runWithTrustRetry(operation);
    } catch (err) {
      if (!parseHostKeyTrustError(err)) {
        message.error(err instanceof Error ? err.message : '保存失败');
      }
    } finally {
      setModalSaving(false);
    }
  };

  const handleDeletePath = async (item: HostFileItem) => {
    if (!id) return;
    try {
      const operation = async () => {
        await Api.hosts.deletePath(id, item.path);
        if (item.path === activeFilePath) {
          setActiveFilePath('');
          setModalContent('');
          setModalDirty(false);
          setFileModalOpen(false);
        }
        await refreshFiles(cwd);
      };
      retryOperationRef.current = operation;
      await runWithTrustRetry(operation);
    } catch (err) {
      if (!parseHostKeyTrustError(err)) {
        message.error(err instanceof Error ? err.message : '删除失败');
      }
    }
  };

  const renamePath = (item: HostFileItem) => {
    if (!id) return;
    let nextName = item.name;
    Modal.confirm({
      title: '重命名',
      content: <Input defaultValue={item.name} onChange={(e) => { nextName = e.target.value; }} />,
      onOk: async () => {
        const parent = item.path.includes('/') ? item.path.slice(0, item.path.lastIndexOf('/')) : '.';
        const operation = async () => {
          await Api.hosts.renamePath(id, item.path, `${parent}/${nextName}`);
          await refreshFiles(cwd);
        };
        retryOperationRef.current = operation;
        try {
          await runWithTrustRetry(operation);
        } catch (err) {
          if (!parseHostKeyTrustError(err)) {
            message.error(err instanceof Error ? err.message : '重命名失败');
          }
        }
      },
    });
  };

  const downloadFile = async (item: HostFileItem) => {
    if (!id || item.is_dir || typeof document === 'undefined') return;
    try {
      const operation = async () => {
        const blob = await Api.hosts.downloadFile(id, item.path);
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = item.name;
        a.click();
        URL.revokeObjectURL(url);
      };
      retryOperationRef.current = operation;
      await runWithTrustRetry(operation);
    } catch (err) {
      if (!parseHostKeyTrustError(err)) {
        message.error(err instanceof Error ? err.message : '下载失败');
      }
    }
  };

  const toParentPath = React.useCallback((path: string) => {
    if (path === '.') return '.';
    if (!path.includes('/')) return '.';
    const parent = path.slice(0, path.lastIndexOf('/'));
    return parent || '.';
  }, []);

  const formatLsTime = React.useCallback((value?: string) => {
    if (!value) return '-';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '-';
    const mm = String(date.getMonth() + 1).padStart(2, '0');
    const dd = String(date.getDate()).padStart(2, '0');
    const hh = String(date.getHours()).padStart(2, '0');
    const min = String(date.getMinutes()).padStart(2, '0');
    return `${mm}-${dd} ${hh}:${min}`;
  }, []);

  const handleCloseFileModal = React.useCallback(() => {
    if (!modalDirty) {
      setFileModalOpen(false);
    }
  }, [modalDirty]);

  const handleConfirmCloseFileModal = React.useCallback(() => {
    setFileModalOpen(false);
  }, []);

  return (
    <div
      className="fade-in host-terminal-page"
      style={{
        flex: 1,
        height: 'calc(100dvh - var(--app-shell-offset, 4rem) - (var(--app-content-y-padding, 1rem) * 2))',
        minHeight: 0,
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
        gap: 4,
      }}
    >
      <Breadcrumb>
        <Breadcrumb.Item><Link to="/deployment/infrastructure/hosts">主机管理</Link></Breadcrumb.Item>
        <Breadcrumb.Item><Link to={`/deployment/infrastructure/hosts/${id}`}>{host?.name || `Host #${id}`}</Link></Breadcrumb.Item>
        <Breadcrumb.Item>终端与文件</Breadcrumb.Item>
      </Breadcrumb>

      <Card
        style={{ borderRadius: 10, flex: 1, minHeight: 0, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}
        styles={{ body: { padding: 12, minHeight: 0, flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' } }}
        title={
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/deployment/infrastructure/hosts/${id}`)}>返回</Button>
            <Text strong>{host?.name || `Host #${id}`}</Text>
            <Text type="secondary">{host?.ip || '-'}</Text>
            <Tag color={status === 'connected' ? 'success' : status === 'connecting' ? 'processing' : status === 'error' ? 'error' : 'default'}>
              {status.toUpperCase()}
            </Tag>
          </Space>
        }
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void connect()}>重连</Button>
            <Button danger onClick={() => void closeSession()}>关闭会话</Button>
          </Space>
        }
      >
        <Row gutter={12} style={{ flex: 1, minHeight: 0 }} align="stretch">
          <Col xs={24} xl={17} style={{ display: 'flex', minHeight: 0, minWidth: 0, overflow: 'hidden' }}>
            <Card
              size="small"
              styles={{ body: { padding: 0, background: '#0e1117', minHeight: 0, flex: 1, display: 'flex', overflow: 'hidden' } }}
              style={{ borderRadius: 10, border: '1px solid #1f2937', width: '100%', minHeight: 0, flex: 1, display: 'flex', flexDirection: 'column' }}
            >
              <div className="host-terminal-xterm" ref={termWrapRef} style={{ height: '100%', width: '100%', minHeight: 0 }} />
            </Card>
          </Col>
          <Col xs={24} xl={7} style={{ display: 'flex', minHeight: 0, minWidth: 0, overflow: 'hidden' }}>
            <div ref={filePaneRef} style={{ width: '100%', minHeight: 0, display: 'flex' }}>
              <Card
                size="small"
                title="文件管理"
                extra={
                  <Space size={4}>
                    <Button size="small" icon={<ReloadOutlined />} onClick={() => void refreshFiles(cwd)} />
                    <Button size="small" icon={<FolderAddOutlined />} onClick={() => setNewDirOpen(true)} />
                    <Upload
                      showUploadList={false}
                      customRequest={async (opt) => {
                        const file = opt.file as File;
                        const operation = async () => {
                          await Api.hosts.uploadFile(id, cwd, file);
                          opt.onSuccess?.({}, new XMLHttpRequest());
                          await refreshFiles(cwd);
                        };
                        retryOperationRef.current = operation;
                        try {
                          await runWithTrustRetry(operation);
                        } catch (err) {
                          if (!parseHostKeyTrustError(err)) {
                            message.error(err instanceof Error ? err.message : '上传失败');
                            opt.onError?.(err as Error);
                          }
                        }
                      }}
                    >
                      <Button size="small" icon={<UploadOutlined />} />
                    </Upload>
                  </Space>
                }
                style={{ borderRadius: 10, minHeight: 0, width: '100%', flex: 1, display: 'flex', flexDirection: 'column' }}
                styles={{
                  header: { minHeight: 40, padding: '0 12px' },
                  body: { padding: '8px 12px 10px', display: 'flex', flexDirection: 'column', gap: 6, minHeight: 0, flex: 1, overflow: 'hidden' },
                }}
              >
              <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                <Text type="secondary">目录: {cwd}</Text>
                <Space.Compact style={{ width: 220 }}>
                  <Input
                    size="small"
                    placeholder="输入目录并跳转"
                    value={pathInput}
                    onChange={(e) => setPathInput(e.target.value)}
                    onPressEnter={() => void refreshFiles((pathInput || '.').trim() || '.')}
                  />
                  <Button size="small" onClick={() => void refreshFiles((pathInput || '.').trim() || '.')}>跳转</Button>
                </Space.Compact>
              </Space>
              {filesLoading ? <Spin /> : null}
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: fileGridColumns,
                  alignItems: 'center',
                  columnGap: 12,
                  fontFamily: 'JetBrains Mono, Menlo, Monaco, Consolas, monospace',
                  fontSize: 12,
                  color: '#8c8c8c',
                  padding: '2px 8px',
                }}
              >
                <span>名称</span>
                {showMTime ? <span>修改时间</span> : null}
                {showSize ? <span style={{ textAlign: 'right' }}>大小</span> : null}
                <span />
              </div>
              <div style={{ width: '100%', overflowY: 'auto', overflowX: 'hidden', flex: 1, minHeight: 0 }}>
                {cwd !== '.' ? (
                  <div
                    style={{ display: 'grid', gridTemplateColumns: fileGridColumns, alignItems: 'center', columnGap: 12, borderRadius: 8, padding: '2px 8px' }}
                  >
                    <div
                      onClick={() => void refreshFiles(toParentPath(cwd))}
                      style={{ cursor: 'pointer', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                    >
                      <span title="..">📁 ..</span>
                    </div>
                    {showMTime ? <span>-</span> : null}
                    {showSize ? <span style={{ textAlign: 'right' }}>-</span> : null}
                    <span />
                  </div>
                ) : null}
                {files.map((item) => (
                  <div
                    key={item.path}
                    style={{
                      display: 'grid',
                      gridTemplateColumns: fileGridColumns,
                      alignItems: 'center',
                      columnGap: 12,
                      borderRadius: 8,
                      padding: '2px 8px',
                      background: activeFilePath === item.path ? '#e6f4ff' : 'transparent',
                    }}
                  >
                    <div
                      onClick={() => void openFile(item)}
                      style={{ cursor: 'pointer', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
                    >
                      <span title={item.name}>
                        {item.is_dir ? '📁' : '📄'} {item.name}
                      </span>
                    </div>
                    {showMTime ? <span>{formatLsTime(item.updated_at)}</span> : null}
                    {showSize ? <span style={{ textAlign: 'right' }}>{item.is_dir ? '-' : String(item.size ?? 0)}</span> : null}
                    <Space size={0} style={{ justifyContent: 'flex-end' }}>
                      {!item.is_dir ? <Button type="text" size="small" icon={<DownloadOutlined />} onClick={() => void downloadFile(item)} /> : null}
                      <Button type="text" size="small" icon={<EditOutlined />} onClick={() => renamePath(item)} />
                      <Popconfirm
                        title={`确定删除 ${item.name}？`}
                        okText="确定"
                        cancelText="取消"
                        okButtonProps={{ danger: true }}
                        onConfirm={() => void handleDeletePath(item)}
                      >
                        <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                      </Popconfirm>
                    </Space>
                  </div>
                ))}
              </div>
              </Card>
            </div>
          </Col>
        </Row>
      </Card>

      <Modal
        open={fileModalOpen}
        title={activeFilePath ? <Text ellipsis={{ tooltip: activeFilePath }} style={{ maxWidth: '100%', display: 'block' }}>{`编辑: ${activeFilePath}`}</Text> : '文件编辑'}
        onCancel={() => {
          if (!modalDirty) {
            setFileModalOpen(false);
          } else {
            Modal.confirm({
              title: '放弃未保存修改？',
              content: '当前文件尚未保存，确认关闭编辑窗口吗？',
              okText: '放弃修改',
              cancelText: '继续编辑',
              onOk: handleConfirmCloseFileModal,
            });
          }
        }}
        width="80vw"
        styles={{ body: { height: '80vh', display: 'flex', flexDirection: 'column', minHeight: 0 } }}
        footer={(
          <Space>
            {modalDirty ? (
              <Popconfirm
                title="放弃未保存修改？"
                okText="放弃"
                cancelText="继续编辑"
                onConfirm={handleConfirmCloseFileModal}
              >
                <Button>取消</Button>
              </Popconfirm>
            ) : (
              <Button onClick={handleCloseFileModal}>取消</Button>
            )}
            <Button type="primary" icon={<SaveOutlined />} loading={modalSaving} onClick={() => void saveFile()}>保存</Button>
          </Space>
        )}
      >
        <div style={{ flex: 1, minHeight: 0 }}>
          <Editor
            height="100%"
            defaultLanguage="yaml"
            value={modalContent}
            onChange={(v) => { setModalContent(v || ''); setModalDirty(true); }}
            theme="vs-dark"
            options={{ minimap: { enabled: false }, fontSize: 13 }}
          />
        </div>
        {modalDirty ? <Alert style={{ marginTop: 8 }} type="warning" showIcon message="内容已修改，记得保存。" /> : null}
      </Modal>

      <Modal
        open={newDirOpen}
        title="新建目录"
        onOk={async () => {
          if (!newDirName.trim()) return;
          const operation = async () => {
            await Api.hosts.mkdir(id, `${cwd}/${newDirName.trim()}`.replace('//', '/'));
            setNewDirOpen(false);
            setNewDirName('');
            await refreshFiles(cwd);
          };
          retryOperationRef.current = operation;
          try {
            await runWithTrustRetry(operation);
          } catch (err) {
            if (!parseHostKeyTrustError(err)) {
              message.error(err instanceof Error ? err.message : '创建目录失败');
            }
          }
        }}
        onCancel={() => setNewDirOpen(false)}
      >
        <Input
          prefix={<FileAddOutlined />}
          placeholder="目录名"
          value={newDirName}
          onChange={(e) => setNewDirName(e.target.value)}
        />
      </Modal>

      <HostKeyTrustModal
        open={Boolean(pendingTrust)}
        loading={confirming}
        mode={pendingTrust?.errorType === 'ssh_host_key_mismatch' ? 'rotate' : 'create'}
        hostKey={pendingTrust?.hostKey || null}
        onCancel={() => setPendingTrust(null)}
        onConfirm={async () => {
          try {
            await confirmTrustAndRetry(async () => {
              await retryOperationRef.current();
            });
          } catch (err) {
            if (!parseHostKeyTrustError(err)) {
              message.error(err instanceof Error ? err.message : '信任主机指纹失败');
            }
          }
        }}
      />
    </div>
  );
};

export default HostTerminalPage;
