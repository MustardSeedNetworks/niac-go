/**
 * ErrorBoundary component for catching and displaying React errors.
 * SECURITY FIX #159: Prevents crashes from exposing internal state and provides graceful degradation.
 */

import { AlertCircle, RefreshCw } from 'lucide-react';
import { Component, type ErrorInfo, type ReactNode } from 'react';
// An error boundary must be a class, so it cannot use the useTranslation
// hook. <Trans> is a component and subscribes to language changes, so the
// fallback still re-renders when the locale switches.
import { Trans } from 'react-i18next';
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
        <div
          data-testid="error-boundary-fallback"
          className="flex min-h-[400px] flex-col items-center justify-center pad-xl"
        >
          <div className="mx-auto max-w-md text-center">
            <div className="mb-content flex justify-center">
              <div className="rounded-full bg-status-error/10 pad">
                <AlertCircle className={`${iconSizes['3xl']} text-status-error`} />
              </div>
            </div>

            <h2 className="mb-2 heading-2 text-text-primary">
              <Trans i18nKey="errorBoundary.title" ns="common" />
            </h2>

            <p className="mb-section text-sm text-text-muted">
              <Trans i18nKey="errorBoundary.description" ns="common" />
            </p>

            {/* Error details (development only - hidden in production) */}
            {import.meta.env.DEV && this.state.error && (
              <div className="mb-section rounded-lg bg-bg-muted pad text-left">
                <p className="mb-tight text-xs font-medium text-text-muted">
                  <Trans i18nKey="errorBoundary.errorDetails" ns="common" />
                </p>
                <pre className="whitespace-pre-wrap break-words text-xs text-status-error">
                  {this.state.error.message}
                </pre>
                {this.state.errorInfo?.componentStack && (
                  <>
                    <p className="mb-tight mt-heading text-xs font-medium text-text-muted">
                      <Trans i18nKey="errorBoundary.componentStack" ns="common" />
                    </p>
                    {/* Not a scroll box: a capped, scrollable pane needs to be
                        keyboard-reachable (axe scrollable-region-focusable), and a
                        tabIndex on a non-interactive element is what Biome's
                        noNoninteractiveTabindex forbids. The stack is DEV-only
                        output, so it simply renders in full. */}
                    <pre className="text-xs text-text-muted">
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
                className="inline-flex items-center gap-compact rounded-lg bg-status-info px-4 py-row text-sm font-medium text-on-info transition-colors hover:bg-status-info/85 focus:outline-none focus:ring-2 focus:ring-status-info focus:ring-offset-2"
              >
                <RefreshCw className={iconSizes.md} />
                <Trans i18nKey="errorBoundary.tryAgain" ns="common" />
              </button>
              <button
                type="button"
                onClick={this.handleReload}
                className="inline-flex items-center gap-compact rounded-lg border border-border-muted bg-bg-surface px-4 py-row text-sm font-medium text-text-secondary transition-colors hover:bg-bg-muted focus:outline-none focus:ring-2 focus:ring-status-info focus:ring-offset-2"
              >
                <Trans i18nKey="errorBoundary.reload" ns="common" />
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
        <div className="rounded-lg border border-status-error/30 bg-status-error/10 pad-lg">
          <div className="flex items-start gap-comfortable">
            <AlertCircle className={`${iconSizes.xl} flex-shrink-0 text-status-error`} />
            <div className="flex-1">
              <h3 className="text-sm font-medium text-status-error-strong">
                <Trans i18nKey="errorBoundary.pageTitle" ns="common" />
              </h3>
              <p className="mt-tight text-sm text-status-error-strong">
                <Trans i18nKey="errorBoundary.pageDescription" ns="common" />
              </p>
              <div className="mt-content flex gap-compact">
                <button
                  type="button"
                  onClick={this.handleRetry}
                  className="text-sm font-medium text-status-error-strong hover:underline"
                >
                  <Trans i18nKey="errorBoundary.retry" ns="common" />
                </button>
                <span className="text-status-error" aria-hidden="true">
                  |
                </span>
                <button
                  type="button"
                  onClick={this.handleReload}
                  className="text-sm font-medium text-status-error hover:underline"
                >
                  <Trans i18nKey="buttons.reload" ns="common" />
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
