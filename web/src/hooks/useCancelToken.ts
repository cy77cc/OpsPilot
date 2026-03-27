import { useEffect, useRef } from 'react';

/**
 * Hook for cancelling API requests when component unmounts
 *
 * @example
 * ```tsx
 * const { getSignal, cancel } = useCancelToken();
 *
 * useEffect(() => {
 *   const fetchData = async () => {
 *     try {
 *       const response = await fetch('/api/data', {
 *         signal: getSignal(),
 *       });
 *       const data = await response.json();
 *       setData(data);
 *     } catch (error) {
 *       if (error.name === 'AbortError') {
 *         console.log('Request cancelled');
 *       }
 *     }
 *   };
 *
 *   fetchData();
 * }, []);
 * ```
 */
export function useCancelToken() {
  const abortControllerRef = useRef<AbortController | null>(null);

  /**
   * Get abort signal for current request
   */
  const getSignal = (): AbortSignal => {
    // Cancel previous request if exists
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }

    // Create new abort controller
    abortControllerRef.current = new AbortController();
    return abortControllerRef.current.signal;
  };

  /**
   * Manually cancel current request
   */
  const cancel = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
  };

  /**
   * Check if request was cancelled
   */
  const isCancelled = (error: any): boolean => {
    return error?.name === 'AbortError' || error?.name === 'CanceledError';
  };

  // Cancel on unmount
  useEffect(() => {
    return () => {
      cancel();
    };
  }, []);

  return {
    getSignal,
    cancel,
    isCancelled,
  };
}

/**
 * Hook for managing multiple concurrent requests with cancellation
 *
 * @example
 * ```tsx
 * const { addRequest, cancelAll } = useRequestManager();
 *
 * const fetchData = async (id: string) => {
 *   const controller = new AbortController();
 *   addRequest(id, controller);
 *
 *   try {
 *     const response = await fetch(`/api/data/${id}`, {
 *       signal: controller.signal,
 *     });
 *     return await response.json();
 *   } catch (error) {
 *     if (error.name === 'AbortError') {
 *       console.log('Request cancelled');
 *     }
 *     throw error;
 *   }
 * };
 * ```
 */
export function useRequestManager() {
  const controllersRef = useRef<Map<string, AbortController>>(new Map());

  /**
   * Add a request to be managed
   */
  const addRequest = (key: string, controller: AbortController) => {
    // Cancel previous request with same key
    const existing = controllersRef.current.get(key);
    if (existing) {
      existing.abort();
    }

    controllersRef.current.set(key, controller);
  };

  /**
   * Cancel a specific request by key
   */
  const cancelRequest = (key: string) => {
    const controller = controllersRef.current.get(key);
    if (controller) {
      controller.abort();
      controllersRef.current.delete(key);
    }
  };

  /**
   * Cancel all managed requests
   */
  const cancelAll = () => {
    controllersRef.current.forEach((controller) => {
      controller.abort();
    });
    controllersRef.current.clear();
  };

  /**
   * Remove a request from management (after completion)
   */
  const removeRequest = (key: string) => {
    controllersRef.current.delete(key);
  };

  // Cancel all on unmount
  useEffect(() => {
    return () => {
      cancelAll();
    };
  }, []);

  return {
    addRequest,
    cancelRequest,
    cancelAll,
    removeRequest,
  };
}
