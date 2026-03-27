import { useCallback, useRef } from 'react';
import { message } from 'antd';
import { isNetworkError, isTimeoutError, getErrorMessage } from '../utils/apiErrorHandler';

interface RetryOptions {
  /**
   * Maximum number of retry attempts
   * @default 3
   */
  maxRetries?: number;

  /**
   * Initial delay in milliseconds before first retry
   * @default 1000
   */
  initialDelay?: number;

  /**
   * Multiplier for exponential backoff
   * @default 2
   */
  backoffMultiplier?: number;

  /**
   * Maximum delay in milliseconds
   * @default 10000
   */
  maxDelay?: number;

  /**
   * Callback to determine if error should be retried
   * Return true to retry, false to fail immediately
   */
  shouldRetry?: (error: any, attempt: number) => boolean;

  /**
   * Callback when retry is attempted
   */
  onRetry?: (error: any, attempt: number, delay: number) => void;
}

/**
 * Default retry logic - retry on network and timeout errors
 */
function defaultShouldRetry(error: any, attempt: number): boolean {
  // Don't retry client errors (4xx) except 408 (timeout) and 429 (rate limit)
  const status = error?.response?.status;
  if (status >= 400 && status < 500 && status !== 408 && status !== 429) {
    return false;
  }

  // Retry network and timeout errors
  return isNetworkError(error) || isTimeoutError(error) || status >= 500;
}

/**
 * Hook for retrying failed API calls with exponential backoff
 *
 * @param options - Retry configuration
 *
 * @example
 * ```tsx
 * const { withRetry } = useRetry({
 *   maxRetries: 3,
 *   initialDelay: 1000,
 * });
 *
 * const fetchData = withRetry(async () => {
 *   const response = await Api.getData();
 *   return response;
 * });
 * ```
 */
export function useRetry(options: RetryOptions = {}) {
  const {
    maxRetries = 3,
    initialDelay = 1000,
    backoffMultiplier = 2,
    maxDelay = 10000,
    shouldRetry = defaultShouldRetry,
    onRetry,
  } = options;

  const optionsRef = useRef({ maxRetries, initialDelay, backoffMultiplier, maxDelay, shouldRetry, onRetry });

  // Update ref when options change
  optionsRef.current = { maxRetries, initialDelay, backoffMultiplier, maxDelay, shouldRetry, onRetry };

  /**
   * Wrap an async function with retry logic
   */
  const withRetry = useCallback(
    async <T,>(fn: () => Promise<T>): Promise<T> => {
      const opts = optionsRef.current;
      let lastError: any;
      let attempt = 0;

      while (attempt <= opts.maxRetries) {
        try {
          return await fn();
        } catch (error) {
          lastError = error;
          attempt++;

          // Check if we should retry
          if (attempt > opts.maxRetries || !opts.shouldRetry(error, attempt)) {
            throw error;
          }

          // Calculate delay with exponential backoff
          const delay = Math.min(
            opts.initialDelay * Math.pow(opts.backoffMultiplier, attempt - 1),
            opts.maxDelay
          );

          // Notify about retry
          opts.onRetry?.(error, attempt, delay);

          console.warn(`API call failed, retrying (${attempt}/${opts.maxRetries}) after ${delay}ms:`, {
            error: getErrorMessage(error),
            attempt,
            delay,
          });

          // Wait before retrying
          await new Promise((resolve) => setTimeout(resolve, delay));
        }
      }

      throw lastError;
    },
    []
  );

  return { withRetry };
}

/**
 * Standalone retry function without hook
 */
export async function retryAsync<T>(
  fn: () => Promise<T>,
  options: RetryOptions = {}
): Promise<T> {
  const {
    maxRetries = 3,
    initialDelay = 1000,
    backoffMultiplier = 2,
    maxDelay = 10000,
    shouldRetry = defaultShouldRetry,
    onRetry,
  } = options;

  let lastError: any;
  let attempt = 0;

  while (attempt <= maxRetries) {
    try {
      return await fn();
    } catch (error) {
      lastError = error;
      attempt++;

      if (attempt > maxRetries || !shouldRetry(error, attempt)) {
        throw error;
      }

      const delay = Math.min(
        initialDelay * Math.pow(backoffMultiplier, attempt - 1),
        maxDelay
      );

      onRetry?.(error, attempt, delay);

      console.warn(`API call failed, retrying (${attempt}/${maxRetries}) after ${delay}ms:`, {
        error: getErrorMessage(error),
        attempt,
        delay,
      });

      await new Promise((resolve) => setTimeout(resolve, delay));
    }
  }

  throw lastError;
}
