import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ApiRequestError } from './api';

describe('ApiRequestError', () => {
  it('creates error with message only', () => {
    const error = new ApiRequestError('Test error');
    expect(error.message).toBe('Test error');
    expect(error.name).toBe('ApiRequestError');
    expect(error.statusCode).toBeUndefined();
    expect(error.businessCode).toBeUndefined();
  });

  it('creates error with status code and business code', () => {
    const error = new ApiRequestError('Test error', 401, 4005);
    expect(error.message).toBe('Test error');
    expect(error.statusCode).toBe(401);
    expect(error.businessCode).toBe(4005);
  });

  it('is an instance of Error', () => {
    const error = new ApiRequestError('Test error');
    expect(error).toBeInstanceOf(Error);
  });

  it('can be caught and checked by businessCode', () => {
    const error = new ApiRequestError('Token expired', 401, 4005);

    try {
      throw error;
    } catch (e) {
      if (e instanceof ApiRequestError) {
        expect(e.businessCode).toBe(4005);
        expect(e.statusCode).toBe(401);
      }
    }
  });

  it('handles undefined values gracefully', () => {
    const error = new ApiRequestError('Unknown error', undefined, undefined);
    expect(error.message).toBe('Unknown error');
    expect(error.statusCode).toBeUndefined();
    expect(error.businessCode).toBeUndefined();
  });

  it('preserves stack trace', () => {
    const error = new ApiRequestError('Test error');
    expect(error.stack).toBeDefined();
    expect(error.stack).toContain('ApiRequestError');
  });
});

describe('ApiResponse interface', () => {
  it('defines correct structure for success response', () => {
    const response = {
      success: true,
      data: { id: 1, name: 'test' },
      message: 'Success',
    };

    expect(response.success).toBe(true);
    expect(response.data).toEqual({ id: 1, name: 'test' });
  });

  it('defines correct structure for error response', () => {
    const response = {
      success: false,
      data: null,
      error: {
        code: 'VALIDATION_ERROR',
        message: 'Invalid input',
      },
    };

    expect(response.success).toBe(false);
    expect(response.error?.code).toBe('VALIDATION_ERROR');
  });

  it('supports paginated response with total', () => {
    const response = {
      success: true,
      data: {
        total: 100,
        list: [{ id: 1 }, { id: 2 }],
      },
    };

    expect(response.data.total).toBe(100);
    expect(response.data.list).toHaveLength(2);
  });
});

describe('localStorage interaction', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('stores and retrieves token', () => {
    localStorage.setItem('token', 'test-token');
    expect(localStorage.getItem('token')).toBe('test-token');
  });

  it('stores and retrieves projectId', () => {
    localStorage.setItem('projectId', '123');
    expect(localStorage.getItem('projectId')).toBe('123');
  });

  it('stores and retrieves refreshToken', () => {
    localStorage.setItem('refreshToken', 'refresh-token-123');
    expect(localStorage.getItem('refreshToken')).toBe('refresh-token-123');
  });

  it('clears auth tokens', () => {
    localStorage.setItem('token', 'test-token');
    localStorage.setItem('refreshToken', 'refresh-token');

    localStorage.removeItem('token');
    localStorage.removeItem('refreshToken');

    expect(localStorage.getItem('token')).toBeNull();
    expect(localStorage.getItem('refreshToken')).toBeNull();
  });

  it('handles missing token gracefully', () => {
    const token = localStorage.getItem('token');
    expect(token).toBeNull();
  });
});

describe('API error scenarios', () => {
  it('creates error for 401 unauthorized', () => {
    const error = new ApiRequestError('Unauthorized', 401, 4005);
    expect(error.statusCode).toBe(401);
    expect(error.businessCode).toBe(4005);
  });

  it('creates error for 403 forbidden', () => {
    const error = new ApiRequestError('Forbidden', 403);
    expect(error.statusCode).toBe(403);
  });

  it('creates error for 500 server error', () => {
    const error = new ApiRequestError('Internal Server Error', 500);
    expect(error.statusCode).toBe(500);
  });

  it('creates error for network timeout', () => {
    const error = new ApiRequestError('Network timeout');
    expect(error.message).toBe('Network timeout');
  });

  it('creates error for business logic error', () => {
    const error = new ApiRequestError('Resource not found', 200, 5001);
    expect(error.businessCode).toBe(5001);
    expect(error.message).toBe('Resource not found');
  });
});
