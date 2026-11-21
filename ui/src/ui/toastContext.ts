import { type Context, createContext } from 'react';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface ToastInput {
  type?: ToastType;
  title: string;
  message?: string;
  /**
   * Structured detail lines from the API's `details[]`. The server already
   * says things like "line 65: field enabled not found in type
   * converter.DhcpServer"; a toast that showed only the generic message
   * made the wizard a dead end.
   */
  details?: string[];
  /** Auto-dismiss in ms; 0 or less keeps the toast until it is dismissed. */
  duration?: number;
}

export interface ToastContextValue {
  addToast: (input: ToastInput) => void;
  removeToast: (id: string) => void;
}

export const ToastContext: Context<ToastContextValue | null> =
  createContext<ToastContextValue | null>(null);
