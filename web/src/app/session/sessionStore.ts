import { authApi, type AuthUser, type LoginParams, type RegisterParams } from '../../api/modules/auth';

export type SessionState = {
  user: AuthUser | null;
  permissions: string[];
  loading: boolean;
  isAuthenticated: boolean;
};

type SessionAuthApi = Pick<typeof authApi, 'login' | 'register' | 'getMe' | 'logout'>;

const initialSessionState: SessionState = {
  user: null,
  permissions: [],
  loading: true,
  isAuthenticated: false,
};

export type SessionStore = ReturnType<typeof createSessionStore>;

export function createSessionStore(api: SessionAuthApi = authApi) {
  let state: SessionState = initialSessionState;
  let bootstrapPromise: Promise<AuthUser | null> | null = null;
  const listeners = new Set<() => void>();

  const setState = (next: SessionState) => {
    state = next;
    listeners.forEach((listener) => listener());
  };

  const clearSession = () => {
    setState({
      user: null,
      permissions: [],
      loading: false,
      isAuthenticated: false,
    });
  };

  const finalizeFromUser = (user: AuthUser) => {
    setState({
      user,
      permissions: user.permissions || [],
      loading: false,
      isAuthenticated: true,
    });
  };

  const bootstrap = async () => {
    if (bootstrapPromise) {
      return bootstrapPromise;
    }

    setState({
      ...state,
      loading: true,
    });

    bootstrapPromise = (async () => {
      try {
        const res = await api.getMe();
        finalizeFromUser(res.data);
        return res.data;
      } catch (error) {
        clearSession();
        throw error;
      } finally {
        bootstrapPromise = null;
      }
    })();

    return bootstrapPromise;
  };

  return {
    getSnapshot: () => state,
    subscribe(listener: () => void) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    clearSession,
    async bootstrap() {
      return bootstrap();
    },
    async login(payload: LoginParams) {
      setState({
        ...state,
        loading: true,
      });
      await api.login(payload);
      return bootstrap();
    },
    async register(payload: RegisterParams) {
      setState({
        ...state,
        loading: true,
      });
      await api.register(payload);
      return bootstrap();
    },
    async logout() {
      try {
        await api.logout();
      } finally {
        clearSession();
      }
    },
  };
}

export const sessionStore = createSessionStore();
