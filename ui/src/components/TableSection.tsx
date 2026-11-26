import type { ReactNode } from 'react';

interface TableSectionProps {
  title: string;
  description?: string | undefined;
  /** Right of the heading: a count chip. */
  aside?: ReactNode;
  children: ReactNode;
  'data-testid'?: string;
}

/**
 * A heading, one line of help and the table it introduces. The shared
 * DataTable draws its own frame, so it stands alone under the heading instead
 * of sitting in a Card (a frame inside a frame is the nesting the audit found).
 */
export function TableSection({
  title,
  description,
  aside,
  children,
  'data-testid': testId,
}: TableSectionProps) {
  return (
    <section className="stack" data-testid={testId}>
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h2 className="text-base font-semibold text-text-primary">{title}</h2>
          {description ? <p className="text-sm text-text-muted">{description}</p> : null}
        </div>
        {aside}
      </div>
      {children}
    </section>
  );
}
