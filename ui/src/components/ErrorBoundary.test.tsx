import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ErrorBoundary, PageErrorBoundary } from './ErrorBoundary';

// Component that throws an error
function ThrowingComponent({ shouldThrow = true }: { shouldThrow?: boolean }) {
  if (shouldThrow) {
    throw new Error('Test render error');
  }
  return <div>Child rendered successfully</div>;
}

// Suppress React error boundary console errors in tests
const originalConsoleError = console.error;

describe('ErrorBoundary', () => {
  beforeEach(() => {
    console.error = (...args: unknown[]) => {
      if (
        typeof args[0] === 'string' &&
        (args[0].includes('Error Boundary') || args[0].includes('The above error'))
      ) {
        return;
      }
      originalConsoleError(...args);
    };
  });

  afterEach(() => {
    console.error = originalConsoleError;
  });

  it('renders children when no error occurs', () => {
    render(
      <ErrorBoundary>
        <div>Safe content</div>
      </ErrorBoundary>,
    );

    expect(screen.getByText('Safe content')).toBeInTheDocument();
  });

  it('renders fallback UI when an error occurs', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>,
    );

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText(/unexpected error occurred/)).toBeInTheDocument();
  });

  it('renders custom fallback when provided', () => {
    render(
      <ErrorBoundary fallback={<div>Custom error message</div>}>
        <ThrowingComponent />
      </ErrorBoundary>,
    );

    expect(screen.getByText('Custom error message')).toBeInTheDocument();
  });

  it('calls onError callback when an error occurs', () => {
    const onError = vi.fn();

    render(
      <ErrorBoundary onError={onError}>
        <ThrowingComponent />
      </ErrorBoundary>,
    );

    expect(onError).toHaveBeenCalledTimes(1);
    expect(onError).toHaveBeenCalledWith(
      expect.any(Error),
      expect.objectContaining({ componentStack: expect.any(String) }),
    );
  });

  it('shows retry button that resets error state', async () => {
    const user = userEvent.setup();

    // We need a component that can toggle whether it throws
    let shouldThrow = true;
    function ConditionalThrow() {
      if (shouldThrow) {
        throw new Error('First render error');
      }
      return <div>Recovered content</div>;
    }

    const { rerender } = render(
      <ErrorBoundary>
        <ConditionalThrow />
      </ErrorBoundary>,
    );

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();

    // Fix the throwing condition
    shouldThrow = false;

    // Click Try Again
    await user.click(screen.getByText('Try Again'));

    // Re-render to pick up the fixed component
    rerender(
      <ErrorBoundary>
        <ConditionalThrow />
      </ErrorBoundary>,
    );

    expect(screen.getByText('Recovered content')).toBeInTheDocument();
  });

  it('shows reload button', () => {
    render(
      <ErrorBoundary>
        <ThrowingComponent />
      </ErrorBoundary>,
    );

    expect(screen.getByText('Reload Page')).toBeInTheDocument();
  });
});

describe('PageErrorBoundary', () => {
  beforeEach(() => {
    console.error = (...args: unknown[]) => {
      if (
        typeof args[0] === 'string' &&
        (args[0].includes('Error Boundary') || args[0].includes('The above error'))
      ) {
        return;
      }
      originalConsoleError(...args);
    };
  });

  afterEach(() => {
    console.error = originalConsoleError;
  });

  it('renders page-level error UI', () => {
    render(
      <PageErrorBoundary>
        <ThrowingComponent />
      </PageErrorBoundary>,
    );

    expect(screen.getByText('Page Error')).toBeInTheDocument();
    expect(screen.getByText(/encountered an error/)).toBeInTheDocument();
  });

  it('shows retry and reload links', () => {
    render(
      <PageErrorBoundary>
        <ThrowingComponent />
      </PageErrorBoundary>,
    );

    expect(screen.getByText('Retry')).toBeInTheDocument();
    expect(screen.getByText('Reload')).toBeInTheDocument();
  });
});
