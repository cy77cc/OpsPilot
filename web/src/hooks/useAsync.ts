import { useState, useCallback, useRef, useEffect } from 'react';

export interface AsyncState<T> {
  data: T | null;
  loading: boolean;
  error: Error | null;
}

export interface UseAsyncOptions<T> {
  /**
   * Initial data value
   */
  initialData?: T;

  /**
   * Whether to execute immediately on mount
   * @default false
   */
  immediate?: boolean;

  /**
   * Callback when request succeeds
   */
  onSuccess?: (data: T) => void;

  /**
   * Callback when request fails
   */
  onError?: (error: Error) => void;

  /**
   * Callback when request completes (success or error)
   */
  onFinally?: () => void;
}

/**
 * Hook for managing async operations with loading states
 *
 * @param asyncFunction - Async function to execute
 * @param options - Configuration options
 *
 * @example
 * ```tsx
 * const { data, loading, error, execute, reset } = useAsync(
 *   async (id: string) => {
 *     const response = await Api.getData(id);
 *     return response;
 *   },
 *   {
 *     immediate: false,
 *     onSuccess: (data) => message.success('Data loaded'),
 *     onError: (error) => message.error(error.message),
 *   }
 * );
 *
 * // Execute manually
 * const handleLoad = () => {
 *   execute('123');
 * };
 * ```
 */
export function useAsync<T, Args extends any[] = []>(
  asyncFunction: (...args: Args) => Promise<T>,
  options: UseAsyncOptions<T> = {}
) {
  const {
    initialData = null,
    immediate = false,
    onSuccess,
    onError,
    onFinally,
  } = options;

  const [state, setState] = useState<AsyncState<T>>({
    data: initialData,
    loading: immediate,
    error: null,
  });

  const mountedRef = useRef(true);
  const asyncFunctionRef = useRef(asyncFunction);
  const optionsRef = useRef({ onSuccess, onError, onFinally });

  // Update refs when dependencies change
  useEffect(() => {
    asyncFunctionRef.current = asyncFunction;
  }, [asyncFunction]);

  useEffect(() => {
    optionsRef.current = { onSuccess, onError, onFinally };
  }, [onSuccess, onError, onFinally]);

  // Track mounted state
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const execute = useCallback(async (...args: Args): Promise<T | undefined> => {
    setState((prev) => ({ ...prev, loading: true, error: null }));

    try {
      const data = await asyncFunctionRef.current(...args);

      // Only update state if component is still mounted
      if (mountedRef.current) {
        setState({ data, loading: false, error: null });
        optionsRef.current.onSuccess?.(data);
      }

      return data;
    } catch (error) {
      const err = error instanceof Error ? error : new Error(String(error));

      // Only update state if component is still mounted
      if (mountedRef.current) {
        setState((prev) => ({ ...prev, loading: false, error: err }));
        optionsRef.current.onError?.(err);
      }

      throw err;
    } finally {
      if (mountedRef.current) {
        optionsRef.current.onFinally?.();
      }
    }
  }, []);

  const reset = useCallback(() => {
    setState({
      data: initialData,
      loading: false,
      error: null,
    });
  }, [initialData]);

  const setData = useCallback((data: T) => {
    setState((prev) => ({ ...prev, data }));
  }, []);

  // Execute immediately if requested
  useEffect(() => {
    if (immediate) {
      execute(...([] as unknown as Args));
    }
  }, [immediate, execute]);

  return {
    ...state,
    execute,
    reset,
    setData,
  };
}

/**
 * Hook for managing multiple async operations
 *
 * @example
 * ```tsx
 * const { states, execute, reset } = useAsyncMultiple({
 *   users: () => Api.getUsers(),
 *   posts: () => Api.getPosts(),
 * });
 *
 * // Execute all
 * await execute();
 *
 * // Execute specific
 * await execute('users');
 *
 * // Access state
 * const { data: users, loading: usersLoading } = states.users;
 * ```
 */
export function useAsyncMultiple<T extends Record<string, () => Promise<any>>>(
  asyncFunctions: T
) {
  type Keys = keyof T;
  type States = {
    [K in Keys]: AsyncState<Awaited<ReturnType<T[K]>>>;
  };

  const initialStates = Object.keys(asyncFunctions).reduce((acc, key) => {
    acc[key as Keys] = { data: null, loading: false, error: null };
    return acc;
  }, {} as States);

  const [states, setStates] = useState<States>(initialStates);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const execute = useCallback(
    async (key?: Keys): Promise<void> => {
      const keysToExecute = key ? [key] : (Object.keys(asyncFunctions) as Keys[]);

      // Set loading state
      setStates((prev) => {
        const next = { ...prev };
        keysToExecute.forEach((k) => {
          next[k] = { ...prev[k], loading: true, error: null };
        });
        return next;
      });

      // Execute all functions
      await Promise.all(
        keysToExecute.map(async (k) => {
          try {
            const data = await asyncFunctions[k]();

            if (mountedRef.current) {
              setStates((prev) => ({
                ...prev,
                [k]: { data, loading: false, error: null },
              }));
            }
          } catch (error) {
            const err = error instanceof Error ? error : new Error(String(error));

            if (mountedRef.current) {
              setStates((prev) => ({
                ...prev,
                [k]: { ...prev[k], loading: false, error: err },
              }));
            }
          }
        })
      );
    },
    [asyncFunctions]
  );

  const reset = useCallback((key?: Keys) => {
    if (key) {
      setStates((prev) => ({
        ...prev,
        [key]: { data: null, loading: false, error: null },
      }));
    } else {
      setStates(initialStates);
    }
  }, [initialStates]);

  return {
    states,
    execute,
    reset,
  };
}
