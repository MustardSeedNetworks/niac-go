/**
 * ErrorBoundary component for catching and displaying React errors.
 * SECURITY FIX #159: Prevents crashes from exposing internal state and provides graceful degradation.
 */

import { AlertCircle, RefreshCw } from 'lucide-react';
import { Component, type ErrorInfo, type ReactNode } from 'react';
import { iconSizes } from '../constants/sizes';
import { reportError } from '../utils/error-reporter';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

/**
 * ErrorBoundary catches JavaScript errors anywhere in the child component tree,
 * logs those errors, and displays a fallback UI instead of crashing the app.
 */
export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    // Update state so the next render shows the fallback UI
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    // Store error info for display
    this.setState({ errorInfo });

    // FIX #190: Report error to error tracking
    reportError(error, 'ErrorBoundary');

    // Call optional error handler for external error tracking (e.g., Sentry)
    this.props.onError?.(error, errorInfo);
  }

  handleRetry = (): void => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
    });
  };

  handleReload = (): void => {
    window.location.reload();
  };

  render(): ReactNode {
    if (this.state.hasError) {
      // Custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default error UI
      return (
        <div className="flex min-h-[400px] flex-col items-center justify-center pad-xl">
          <div className="mx-auto max-w-md text-center">
            <div className="mb-content flex justify-center">
              <div className="rounded-full bg-status-error pad dark:bg-status-error/20">
                <AlertCircle
                  className={`${iconSizes['3xl']} text-status-error dark:text-status-error`}
                />
              </div>
            </div>

            <h2 className="mb-2 heading-2 text-text-muted dark:text-text-primary">
              Something went wrong
            </h2>

            <p className="mb-section text-sm text-text-disabled dark:text-text-muted">
              An unexpected error occurred. You can try to recover or reload the page.
            </p>

            {/* Error details (development only - hidden in production) */}
            {import.meta.env.DEV && this.state.error && (
              <div className="mb-section rounded-lg bg-bg-muted pad text-left dark:bg-bg-elevated">
                <p className="mb-tight text-xs font-medium text-text-muted dark:text-text-muted">
                  Error Details:
                </p>
                <pre className="overflow-auto text-xs text-status-error dark:text-status-error">
                  {this.state.error.message}
                </pre>
                {this.state.errorInfo?.componentStack && (
                  <>
                    <p className="mb-tight mt-heading text-xs font-medium text-text-muted dark:text-text-muted">
                      Component Stack:
                    </p>
                    <pre className="max-h-32 overflow-auto text-xs text-text-disabled dark:text-text-muted">
                      {this.state.errorInfo.componentStack}
                    </pre>
                  </>
                )}
              </div>
            )}

            <div className="flex justify-center gap-default">
              <button
                type="button"
                onClick={this.handleRetry}
                className="inline-flex items-center gap-compact rounded-lg bg-status-info px-4 py-row label transition-colors hover:bg-status-info focus:outline-none focus:ring-2 focus:ring-status-info focus:ring-offset-2"
              >
                <RefreshCw className={iconSizes.md} />
                Try Again
              </button>
              <button
                type="button"
                onClick={this.handleReload}
                className="inline-flex items-center gap-compact rounded-lg border border-border-muted bg-white px-4 py-row text-sm font-medium text-text-disabled transition-colors hover:bg-bg-muted focus:outline-none focus:ring-2 focus:ring-status-info focus:ring-offset-2 dark:border-border-muted dark:bg-bg-elevated dark:text-text-secondary dark:hover:bg-bg-elevated"
              >
                Reload Page
              </button>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

/**
 * PageErrorBoundary - A specialized error boundary for page-level errors.
 * Shows a less intrusive error message suitable for individual page failures.
 */
export class PageErrorBoundary extends ErrorBoundary {
  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <div className="rounded-lg border border-status-error bg-status-error pad-lg dark:border-status-error dark:bg-status-error/10">
          <div className="flex items-start gap-comfortable">
            <AlertCircle
              className={`${iconSizes.xl} flex-shrink-0 text-status-error dark:text-status-error`}
            />
            <div className="flex-1">
              <h3 className="text-sm font-medium text-status-error dark:text-status-error">
                Page Error
              </h3>
              <p className="mt-tight text-sm text-status-error dark:text-status-error">
                This page encountered an error. Try navigating to another page or refresh.
              </p>
              <div className="mt-content flex gap-compact">
                <button
                  type="button"
                  onClick={this.handleRetry}
                  className="text-sm font-medium text-status-error hover:text-status-error dark:text-status-error dark:hover:text-status-error"
                >
                  Retry
                </button>
                <span className="text-status-error dark:text-status-error">|</span>
                <button
                  type="button"
                  onClick={this.handleReload}
                  className="text-sm font-medium text-status-error hover:text-status-error dark:text-status-error dark:hover:text-status-error"
                >
                  Reload
                </button>
              </div>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
