import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { type ReactNode, useEffect, useState } from 'react';
import { ToastProvider } from '../ui/Toast';

/**
 * The data layer plus the surface its failures are reported on:
 * useApiResource raises toasts, so wherever resources are available the
 * ToastProvider has to be too.
 */
export function ResourceProvider({ children }: { children: ReactNode }) {
  const [client] = useState(() => new QueryClient());
  useEffect(() => () => client.clear(), [client]);
  return (
    <QueryClientProvider client={client}>
      <ToastProvider>{children}</ToastProvider>
    </QueryClientProvider>
  );
}
