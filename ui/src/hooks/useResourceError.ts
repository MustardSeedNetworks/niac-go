import { useEffect, useRef } from 'react';
import { useErrorToast } from './useErrorToast';

export function useResourceError(error: Error | null, title?: string) {
  const lastMessage = useRef<string | null>(null);
  const showError = useErrorToast();
  useEffect(() => {
    if (!error) {
      lastMessage.current = null;
      return;
    }
    if (lastMessage.current === error.message) return;
    lastMessage.current = error.message;
    showError(error, title);
  }, [error, title, showError]);
}
