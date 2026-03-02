import { useEffect, useRef, useCallback } from 'react';

interface UsePollingOptions {
  /**
   * Polling interval in milliseconds
   * @default 10000 (10 seconds)
   */
  interval?: number;

  /**
   * Whether polling is enabled
   * @default true
   */
  enabled?: boolean;

  /**
   * Callback to determine if polling should stop
   * Return true to stop polling
   */
  shouldStop?: () => boolean;

  /**
   * Callback when polling stops
   */
  onStop?: () => void;
}

/**
 * Hook for polling data at regular intervals
 *
 * @param callback - Function to call on each poll
 * @param options - Polling configuration
 *
 * @example
 * ```tsx
 * const { start, stop, isPolling } = usePolling(
 *   async () => {
 *     const data = await fetchData();
 *     setData(data);
 *   },
 *   {
 *     interval: 5000,
 *     shouldStop: () => data?.status === 'completed',
 *   }
 * );
 * ```
 */
export function usePolling(
  callback: () => void | Promise<void>,
  options: UsePollingOptions = {}
) {
  const {
    interval = 10000,
    enabled = true,
    shouldStop,
    onStop,
  } = options;

  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const isPollingRef = useRef(false);
  const callbackRef = useRef(callback);
  const shouldStopRef = useRef(shouldStop);
  const onStopRef = useRef(onStop);

  // Update refs when dependencies change
  useEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  useEffect(() => {
    shouldStopRef.current = shouldStop;
  }, [shouldStop]);

  useEffect(() => {
    onStopRef.current = onStop;
  }, [onStop]);

  const stop = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
      isPollingRef.current = false;
      onStopRef.current?.();
    }
  }, []);

  const start = useCallback(() => {
    if (intervalRef.current) {
      return; // Already polling
    }

    isPollingRef.current = true;

    const poll = async () => {
      try {
        // Check if should stop before polling
        if (shouldStopRef.current?.()) {
          stop();
          return;
        }

        await callbackRef.current();

        // Check if should stop after polling
        if (shouldStopRef.current?.()) {
          stop();
        }
      } catch (error) {
        console.error('Polling error:', error);
      }
    };

    // Start polling
    intervalRef.current = setInterval(poll, interval);
  }, [interval, stop]);

  // Auto-start/stop based on enabled flag
  useEffect(() => {
    if (enabled) {
      start();
    } else {
      stop();
    }

    // Cleanup on unmount
    return () => {
      stop();
    };
  }, [enabled, start, stop]);

  return {
    start,
    stop,
    isPolling: isPollingRef.current,
  };
}
