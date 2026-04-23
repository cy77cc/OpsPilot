import React, { createContext, useContext, useLayoutEffect, useMemo, useRef, useSyncExternalStore } from 'react';
import type {
  UnreadCountResponse,
  UserNotification,
  WSConnectionStatus,
} from '../types/notification';
import { NotificationDataProvider, useNotificationDataActionsContext, useNotificationDataContext } from './notification/NotificationDataProvider';
import { NotificationWSProvider, useNotificationWSContext } from './notification/NotificationWSProvider';
import { ApprovalStateProvider, type ApprovalActionState, useApprovalStateContext } from './notification/ApprovalStateProvider';

export interface NotificationContextValue {
  notifications: UserNotification[];
  unreadCount: UnreadCountResponse;
  loading: boolean;
  wsStatus: WSConnectionStatus;
  approvalActionStates: Record<string, ApprovalActionState>;
  refresh: () => Promise<void>;
  markAsRead: (id: string) => Promise<void>;
  dismiss: (id: string) => Promise<void>;
  confirm: (id: string) => Promise<void>;
  reject: (id: string) => Promise<void>;
  markAllAsRead: () => Promise<void>;
}

type NotificationContextKey = keyof NotificationContextValue;

interface NotificationContextStore {
  getSnapshot: () => NotificationContextValue;
  setSnapshot: (snapshot: NotificationContextValue) => void;
  subscribe: (
    listener: () => void,
    trackedKeysRef: React.MutableRefObject<Set<NotificationContextKey>>,
  ) => () => void;
}

type NotificationContextSubscriber = {
  listener: () => void;
  trackedKeysRef: React.MutableRefObject<Set<NotificationContextKey>>;
};

const NotificationContextStoreContext = createContext<NotificationContextStore | null>(null);

function shouldNotifySubscriber(
  previousSnapshot: NotificationContextValue,
  nextSnapshot: NotificationContextValue,
  trackedKeys: Set<NotificationContextKey>,
): boolean {
  if (trackedKeys.size === 0) {
    return true;
  }

  for (const key of trackedKeys) {
    if (!Object.is(previousSnapshot[key], nextSnapshot[key])) {
      return true;
    }
  }

  return false;
}

function createNotificationContextStore(initialSnapshot: NotificationContextValue): NotificationContextStore {
  let currentSnapshot = initialSnapshot;
  const subscribers = new Set<NotificationContextSubscriber>();

  return {
    getSnapshot: () => currentSnapshot,
    setSnapshot: (snapshot) => {
      if (Object.is(currentSnapshot, snapshot)) {
        return;
      }

      const previousSnapshot = currentSnapshot;
      currentSnapshot = snapshot;

      subscribers.forEach(({ listener, trackedKeysRef }) => {
        if (shouldNotifySubscriber(previousSnapshot, snapshot, trackedKeysRef.current)) {
          listener();
        }
      });
    },
    subscribe: (listener, trackedKeysRef) => {
      const subscriber = { listener, trackedKeysRef };
      subscribers.add(subscriber);
      return () => {
        subscribers.delete(subscriber);
      };
    },
  };
}

function LegacyNotificationContextProvider({ children }: { children: React.ReactNode }) {
  const { notifications, unreadCount, loading } = useNotificationDataContext();
  const { refresh, markAsRead, dismiss, markAllAsRead } = useNotificationDataActionsContext();
  const { wsStatus } = useNotificationWSContext();
  const { approvalActionStates, confirm, reject } = useApprovalStateContext();
  const storeRef = useRef<NotificationContextStore | null>(null);

  const snapshot = useMemo<NotificationContextValue>(() => ({
    notifications,
    unreadCount,
    loading,
    wsStatus,
    approvalActionStates,
    refresh,
    markAsRead,
    dismiss,
    confirm,
    reject,
    markAllAsRead,
  }), [
    approvalActionStates,
    confirm,
    dismiss,
    loading,
    markAllAsRead,
    markAsRead,
    notifications,
    refresh,
    reject,
    unreadCount,
    wsStatus,
  ]);

  if (!storeRef.current) {
    storeRef.current = createNotificationContextStore(snapshot);
  }

  useLayoutEffect(() => {
    storeRef.current?.setSnapshot(snapshot);
  }, [snapshot]);

  return (
    <NotificationContextStoreContext.Provider value={storeRef.current}>
      {children}
    </NotificationContextStoreContext.Provider>
  );
}

interface NotificationProviderProps {
  children: React.ReactNode;
  userId?: number | string;
  pollInterval?: number;
}

export const NotificationProvider: React.FC<NotificationProviderProps> = ({
  children,
  userId,
  pollInterval = 30000,
}) => {
  return (
    <NotificationDataProvider userId={userId}>
      <NotificationWSProvider userId={userId} pollInterval={pollInterval}>
        <ApprovalStateProvider>
          <LegacyNotificationContextProvider>{children}</LegacyNotificationContextProvider>
        </ApprovalStateProvider>
      </NotificationWSProvider>
    </NotificationDataProvider>
  );
};

export const useNotificationContext = (): NotificationContextValue => {
  const store = useContext(NotificationContextStoreContext);
  const trackedKeysRef = useRef<Set<NotificationContextKey>>(new Set());

  if (!store) {
    throw new Error('useNotificationContext must be used within NotificationProvider');
  }

  const snapshot = useSyncExternalStore(
    (listener) => store.subscribe(listener, trackedKeysRef),
    store.getSnapshot,
    store.getSnapshot,
  );

  const trackedKeys = new Set<NotificationContextKey>();
  trackedKeysRef.current = trackedKeys;

  return useMemo(
    () => new Proxy(snapshot, {
      get(target, property, receiver) {
        if (typeof property === 'string' && property in target) {
          trackedKeys.add(property as NotificationContextKey);
        }
        return Reflect.get(target, property, receiver);
      },
    }) as NotificationContextValue,
    [snapshot, trackedKeys],
  );
};

export default useNotificationContext;
