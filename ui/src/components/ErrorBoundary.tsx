/**
 * ErrorBoundary component for catching and displaying React errors.
 * SECURITY FIX #159: Prevents crashes from exposing internal state and provides graceful degradation.
 */
import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertCircle, RefreshCw } from "lucide-react";

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
    // Log error to console for debugging
    console.error("[ErrorBoundary] Caught error:", error);
    console.error("[ErrorBoundary] Component stack:", errorInfo.componentStack);

    // Store error info for display
    this.setState({ errorInfo });

    // Call optional error handler
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
        <div className="flex min-h-[400px] flex-col items-center justify-center p-8">
          <div className="mx-auto max-w-md text-center">
            <div className="mb-4 flex justify-center">
              <div className="rounded-full bg-red-100 p-4 dark:bg-red-900/20">
                <AlertCircle className="h-12 w-12 text-red-600 dark:text-red-400" />
              </div>
            </div>

            <h2 className="mb-2 text-xl font-semibold text-gray-900 dark:text-gray-100">
              Something went wrong
            </h2>

            <p className="mb-6 text-sm text-gray-600 dark:text-gray-400">
              An unexpected error occurred. You can try to recover or reload the page.
            </p>

            {/* Error details (development only - hidden in production) */}
            {import.meta.env.DEV && this.state.error && (
              <div className="mb-6 rounded-lg bg-gray-100 p-4 text-left dark:bg-gray-800">
                <p className="mb-1 text-xs font-medium text-gray-500 dark:text-gray-400">
                  Error Details:
                </p>
                <pre className="overflow-auto text-xs text-red-600 dark:text-red-400">
                  {this.state.error.message}
                </pre>
                {this.state.errorInfo?.componentStack && (
                  <>
                    <p className="mb-1 mt-3 text-xs font-medium text-gray-500 dark:text-gray-400">
                      Component Stack:
                    </p>
                    <pre className="max-h-32 overflow-auto text-xs text-gray-600 dark:text-gray-400">
                      {this.state.errorInfo.componentStack}
                    </pre>
                  </>
                )}
              </div>
            )}

            <div className="flex justify-center gap-3">
              <button
                onClick={this.handleRetry}
                className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              >
                <RefreshCw className="h-4 w-4" />
                Try Again
              </button>
              <button
                onClick={this.handleReload}
                className="inline-flex items-center gap-2 rounded-lg border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"
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
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 dark:border-red-800 dark:bg-red-900/10">
          <div className="flex items-start gap-4">
            <AlertCircle className="h-6 w-6 flex-shrink-0 text-red-600 dark:text-red-400" />
            <div className="flex-1">
              <h3 className="text-sm font-medium text-red-800 dark:text-red-200">
                Page Error
              </h3>
              <p className="mt-1 text-sm text-red-700 dark:text-red-300">
                This page encountered an error. Try navigating to another page or refresh.
              </p>
              <div className="mt-4 flex gap-2">
                <button
                  onClick={this.handleRetry}
                  className="text-sm font-medium text-red-600 hover:text-red-500 dark:text-red-400 dark:hover:text-red-300"
                >
                  Retry
                </button>
                <span className="text-red-300 dark:text-red-600">|</span>
                <button
                  onClick={this.handleReload}
                  className="text-sm font-medium text-red-600 hover:text-red-500 dark:text-red-400 dark:hover:text-red-300"
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
