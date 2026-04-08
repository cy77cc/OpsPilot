import { useRef } from 'react';

/**
 * Options for useStableFetch hook
 */
export interface UseStableFetchOptions {
  /**
   * Whether to enable deduplication (default: true)
   * Set to false in tests or when you want to allow concurrent requests
   */
  dedupe?: boolean;
}

/**
 * A hook that prevents duplicate async function calls.
 *
 * This is particularly useful with React 18's StrictMode, which intentionally
 * double-invokes effects in development to help identify side effects.
 *
 * IMPORTANT: The returned function is guaranteed to be stable (never changes),
 * so it's safe to use in useEffect dependencies without causing infinite loops.
 *
 * @example
 * ```tsx
 * const load = useStableFetch(async () => {
 *   const response = await api.get('/data');
 *   setData(response.data);
 * });
 *
 * useEffect(() => {
 *   load();
 * }, [load]); // load is stable, won't cause infinite loop
 * ```
 */
export function useStableFetch(
  fetchFn: () => Promise<void>,
  options: UseStableFetchOptions = {}
): () => void {
  // Store fetchFn and options in refs to ensure stable function reference
  const fetchFnRef = useRef(fetchFn);
  const dedupeRef = useRef(options.dedupe ?? true);
  const inFlightRef = useRef(false);

  // Update refs on every render
  fetchFnRef.current = fetchFn;
  dedupeRef.current = options.dedupe ?? true;

  // Create a stable function that never changes
  const stableFetchRef = useRef<() => void>(() => {
    if (dedupeRef.current && inFlightRef.current) {
      return;
    }

    inFlightRef.current = true;
    fetchFnRef.current().finally(() => {
      inFlightRef.current = false;
    });
  });

  return stableFetchRef.current;
}

export default useStableFetch;