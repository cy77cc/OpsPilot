import { message } from 'antd';

export interface ApiError {
  code?: number;
  message: string;
  details?: unknown;
}

/**
 * Extract error message from various error formats
 */
export function getErrorMessage(error: unknown): string {
  const apiError = error as {
    response?: {
      status?: number;
      data?: { message?: string; msg?: string };
    };
    code?: string;
    message?: string;
    msg?: string;
  };

  if (typeof error === 'string') {
    return error;
  }

  if (error instanceof Error) {
    return error.message;
  }

  if (apiError?.response?.data?.message) {
    return apiError.response.data.message;
  }

  if (apiError?.response?.data?.msg) {
    return apiError.response.data.msg;
  }

  if (apiError?.message) {
    return apiError.message;
  }

  if (apiError?.msg) {
    return apiError.msg;
  }

  return '操作失败，请稍后重试';
}

/**
 * Handle API errors with user-friendly messages
 */
export function handleApiError(error: unknown, context?: string): void {
  const errorMessage = getErrorMessage(error);
  const fullMessage = context ? `${context}: ${errorMessage}` : errorMessage;

  message.error(fullMessage);
}

/**
 * Check if error is a network error
 */
export function isNetworkError(error: unknown): boolean {
  const apiError = error as { message?: string; code?: string; response?: unknown };
  return Boolean(
    apiError?.message === 'Network Error' ||
    apiError?.code === 'ECONNABORTED' ||
    apiError?.code === 'ERR_NETWORK' ||
    !apiError?.response
  );
}

/**
 * Check if error is a timeout error
 */
export function isTimeoutError(error: unknown): boolean {
  const apiError = error as { code?: string; message?: string };
  return Boolean(
    apiError?.code === 'ECONNABORTED' ||
    apiError?.message?.includes('timeout')
  );
}

/**
 * Check if error is an authentication error
 */
export function isAuthError(error: unknown): boolean {
  const apiError = error as { response?: { status?: number } };
  return apiError?.response?.status === 401;
}

/**
 * Check if error is a permission error
 */
export function isPermissionError(error: unknown): boolean {
  const apiError = error as { response?: { status?: number } };
  return apiError?.response?.status === 403;
}

/**
 * Check if error is a not found error
 */
export function isNotFoundError(error: unknown): boolean {
  const apiError = error as { response?: { status?: number } };
  return apiError?.response?.status === 404;
}

/**
 * Check if error is a server error
 */
export function isServerError(error: unknown): boolean {
  const apiError = error as { response?: { status?: number } };
  const status = apiError?.response?.status;
  if (typeof status !== 'number') {
    return false;
  }
  return status >= 500 && status < 600;
}

/**
 * Handle long-running operation errors with specific messages
 */
export function handleLongRunningError(
  error: unknown,
  operation: string
): void {
  if (isTimeoutError(error)) {
    message.warning(`${operation}超时，操作可能仍在后台执行，请稍后刷新查看结果`);
  } else if (isNetworkError(error)) {
    message.error(`${operation}失败：网络连接错误，请检查网络后重试`);
  } else if (isServerError(error)) {
    message.error(`${operation}失败：服务器错误，请稍后重试`);
  } else {
    handleApiError(error, operation);
  }
}
