import type { FC } from 'react';
import type { ApiErrorDetail } from '../api/errors';
import { SmallText } from './Typography';

interface ApiErrorMessageProps {
  message: string;
  /** Structured per-field detail the server sent, when it sent any. */
  details?: readonly ApiErrorDetail[];
}

/**
 * Inline error text plus whatever the server said about *why*.
 *
 * The API enumerates its failures one per offending field — preflight
 * validation (#1461), walk capture (#1488) — and each inline surface used to
 * keep only `err.message` and throw the rest away, so the diagnosis travelled
 * the whole way over the wire and died at the last step (#1472, #1499). One
 * component so the next inline error surface renders the detail for free.
 */
export const ApiErrorMessage: FC<ApiErrorMessageProps> = ({ message, details = [] }) => (
  <div className="text-status-error" role="alert">
    <SmallText className="text-status-error">{message}</SmallText>
    {details.length > 0 && (
      <ul className="mt-tight list-disc pl-5 text-sm text-status-error">
        {details.map((detail) => (
          <li key={`${detail.field ?? ''}-${detail.issue}`}>
            {detail.field ? `${detail.field}: ${detail.issue}` : detail.issue}
          </li>
        ))}
      </ul>
    )}
  </div>
);
