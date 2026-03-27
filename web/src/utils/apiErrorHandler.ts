import { message } from 'antd';

export interface ApiError {
  code?: number;
  message: string;
  details?: any;
}

/**
 * Extract error message from various error formats
 */
export function getErrorMessage(error: any): string {
  if (typeof error === 'string') {
    return error;
  }

  if (error instanceof Error) {
    return error.message;
  }

  if (error?.response?.data?.message) {
    return error.response.data.message;
  }

  if (error?.response?.data?.msg) {
    return error.response.data.msg;
  }

  if (error?.message) {
    return error.message;
  }

  if (error?.msg) {
    return error.msg;
  }

  return '操作失败，请稍后重试';
}

/**
 * Handle API errors with user-friendly messages
 */
export function handleApiError(error: any, context?: string): void {
  const errorMessage = getErrorMessage(error);
  const fullMessage = context ? `${context}: ${errorMessage}` : errorMessage;

  console.error('API Error:', {
    context,
    error,
    message: errorMessage,
  });

  message.error(fullMessage);
}

/**
 * Check if error is a network error
 */
export function isNetworkError(error: any): boolean {
  return (
    error?.message === 'Network Error' ||
    error?.code === 'ECONNABORTED' ||
    error?.code === 'ERR_NETWORK' ||
    !error?.response
  );
}

/**
 * Check if error is a timeout error
 */
export function isTimeoutError(error: any): boolean {
  return (
    error?.code === 'ECONNABORTED' ||
    error?.message?.includes('timeout')
  );
}

/**
 * Check if error is an authentication error
 */
export function isAuthError(error: any): boolean {
  return error?.response?.status === 401;
}

/**
 * Check if error is a permission error
 */
export function isPermissionError(error: any): boolean {
  return error?.response?.status === 403;
}

/**
 * Check if error is a not found error
 */
export function isNotFoundError(error: any): boolean {
  return error?.response?.status === 404;
}

/**
 * Check if error is a server error
 */
export function isServerError(error: any): boolean {
  const status = error?.response?.status;
  return status >= 500 && status < 600;
}

/**
 * Handle long-running operation errors with specific messages
 */
export function handleLongRunningError(
  error: any,
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
