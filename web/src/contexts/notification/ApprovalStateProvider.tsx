import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { aiApi } from '../../api/modules/ai';
import { notificationApi } from '../../api/modules/notification';
import type { UserNotification } from '../../types/notification';
import {
  useNotificationDataActionsContext,
  useNotificationDataContext,
  useNotificationDataMutationsContext,
} from './NotificationDataProvider';

export type ApprovalActionState = {
  state: 'submitting' | 'refresh-needed';
  message?: string;
};

interface ApprovalStateProviderProps {
  children: React.ReactNode;
}

interface ApprovalStateContextValue {
  approvalActionStates: Record<string, ApprovalActionState>;
  confirm: (id: string) => Promise<void>;
  reject: (id: string) => Promise<void>;
}

const ApprovalStateContext = createContext<ApprovalStateContextValue | null>(null);

function getApprovalActionKey(notification: UserNotification): string {
  return notification.notification.source_id || notification.id;
}

function getApprovalErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  if (typeof error === 'string' && error.trim()) {
    return error.trim();
  }
  return '提交失败，请刷新后重试';
}

function createApprovalIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }

  return `approval-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export const ApprovalStateProvider: React.FC<ApprovalStateProviderProps> = ({
  children,
}) => {
  const mountedRef = useRef(false);
  const approvalActionInflightRef = useRef<Record<string, string>>({});
  const [approvalActionStates, setApprovalActionStates] = useState<Record<string, ApprovalActionState>>({});
  const { notifications } = useNotificationDataContext();
  const { refresh, dismiss } = useNotificationDataActionsContext();
  const { confirmNotificationLocally, removeNotificationLocally } = useNotificationDataMutationsContext();

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const clearApprovalActionState = useCallback((key: string) => {
    if (!mountedRef.current) {
      return;
    }

    setApprovalActionStates((prev) => {
      if (!prev[key]) {
        return prev;
      }

      const next = { ...prev };
      delete next[key];
      return next;
    });
  }, []);

  const setApprovalActionSubmitting = useCallback((key: string) => {
    if (!mountedRef.current) {
      return;
    }

    setApprovalActionStates((prev) => ({
      ...prev,
      [key]: {
        state: 'submitting',
        message: '提交中',
      },
    }));
  }, []);

  const setApprovalActionFailure = useCallback((key: string, error: unknown) => {
    if (!mountedRef.current) {
      return;
    }

    setApprovalActionStates((prev) => ({
      ...prev,
      [key]: {
        state: 'refresh-needed',
        message: `提交失败：${getApprovalErrorMessage(error)}`,
      },
    }));
  }, []);

  const dispatchApprovalUpdate = useCallback((token: string, status: 'approved' | 'rejected') => {
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent('ai-approval-updated', { detail: { token, status } }));
    }
  }, []);

  const confirm = useCallback(async (id: string) => {
    if (!mountedRef.current) {
      return;
    }

    const target = notifications.find((notification) => notification.id === id);
    if (!target) {
      return;
    }

    if (target.notification.type === 'approval' && target.notification.source_id) {
      const approvalKey = getApprovalActionKey(target);
      if (approvalActionInflightRef.current[approvalKey]) {
        return;
      }

      const idempotencyKey = createApprovalIdempotencyKey();
      approvalActionInflightRef.current[approvalKey] = idempotencyKey;
      setApprovalActionSubmitting(approvalKey);

      try {
        const confirmApproval = (aiApi as unknown as {
          confirmApproval?: (approvalId: string, approved: boolean) => Promise<unknown>;
        }).confirmApproval;

        if (typeof confirmApproval === 'function') {
          await confirmApproval(target.notification.source_id, true);
        } else {
          await (aiApi.submitApproval as unknown as (
            approvalId: string,
            payload: { approved: boolean },
            options?: { idempotencyKey?: string },
          ) => Promise<unknown>)(target.notification.source_id, { approved: true }, {
            idempotencyKey,
          });
        }
      } catch (error) {
        setApprovalActionFailure(approvalKey, error);
        return;
      } finally {
        delete approvalActionInflightRef.current[approvalKey];
      }

      const confirmed = confirmNotificationLocally(id);
      if (confirmed?.notification.source_id) {
        clearApprovalActionState(approvalKey);
        dispatchApprovalUpdate(confirmed.notification.source_id, 'approved');
      }

      return;
    }

    await notificationApi.confirm(id);
    confirmNotificationLocally(id);
    await refresh();
  }, [
    clearApprovalActionState,
    confirmNotificationLocally,
    dispatchApprovalUpdate,
    notifications,
    refresh,
    setApprovalActionFailure,
    setApprovalActionSubmitting,
  ]);

  const reject = useCallback(async (id: string) => {
    if (!mountedRef.current) {
      return;
    }

    const target = notifications.find((notification) => notification.id === id);
    if (!target) {
      return;
    }

    if (target.notification.type === 'approval' && target.notification.source_id) {
      const approvalKey = getApprovalActionKey(target);
      if (approvalActionInflightRef.current[approvalKey]) {
        return;
      }

      const idempotencyKey = createApprovalIdempotencyKey();
      approvalActionInflightRef.current[approvalKey] = idempotencyKey;
      setApprovalActionSubmitting(approvalKey);

      try {
        const confirmApproval = (aiApi as unknown as {
          confirmApproval?: (approvalId: string, approved: boolean) => Promise<unknown>;
        }).confirmApproval;

        if (typeof confirmApproval === 'function') {
          await confirmApproval(target.notification.source_id, false);
        } else {
          await (aiApi.submitApproval as unknown as (
            approvalId: string,
            payload: { approved: boolean },
            options?: { idempotencyKey?: string },
          ) => Promise<unknown>)(target.notification.source_id, { approved: false }, {
            idempotencyKey,
          });
        }
      } catch (error) {
        setApprovalActionFailure(approvalKey, error);
        return;
      } finally {
        delete approvalActionInflightRef.current[approvalKey];
      }

      const removed = removeNotificationLocally(id);
      clearApprovalActionState(approvalKey);
      if (removed?.notification.source_id) {
        dispatchApprovalUpdate(removed.notification.source_id, 'rejected');
      }
      return;
    }

    await dismiss(id);
  }, [
    clearApprovalActionState,
    dismiss,
    dispatchApprovalUpdate,
    notifications,
    removeNotificationLocally,
    setApprovalActionFailure,
    setApprovalActionSubmitting,
  ]);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return undefined;
    }

    const handler = () => {
      void refresh();
    };

    window.addEventListener('ai-approval-updated', handler);
    return () => {
      window.removeEventListener('ai-approval-updated', handler);
    };
  }, [refresh]);

  const value = useMemo<ApprovalStateContextValue>(() => ({
    approvalActionStates,
    confirm,
    reject,
  }), [approvalActionStates, confirm, reject]);

  return (
    <ApprovalStateContext.Provider value={value}>
      {children}
    </ApprovalStateContext.Provider>
  );
};

export function useApprovalStateContext(): ApprovalStateContextValue {
  const context = useContext(ApprovalStateContext);
  if (!context) {
    throw new Error('useApprovalStateContext must be used within ApprovalStateProvider');
  }
  return context;
}
