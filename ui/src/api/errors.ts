/**
 * API Error Types
 *
 * Provides typed error classes and type guards for consistent
 * error handling across the frontend application.
 *
 * FIX #177: Inconsistent error handling
 */

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(message: string, status: number, code = 'API_ERROR') {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

export class NetworkError extends ApiError {
  constructor(message = 'Network error: Unable to reach the server') {
    super(message, 0, 'NETWORK_ERROR');
    this.name = 'NetworkError';
  }
}

export class TimeoutError extends ApiError {
  constructor(message = 'Request timeout') {
    super(message, 0, 'TIMEOUT_ERROR');
    this.name = 'TimeoutError';
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}
