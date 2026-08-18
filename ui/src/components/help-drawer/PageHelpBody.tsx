/**
 * PageHelpBody Component
 *
 * Renders a page's help blocks (see `/src/data/page-help.ts`). Content is
 * data, not JSX, so the drawer can search it — which means the small amount
 * of emphasis the prose needs travels as inline markers: `code`, **strong**,
 * and [label](https://example.com). Anything else is literal text.
 *
 * Links are authored in this repo, never user input.
 */

import type { ReactElement, ReactNode } from 'react';
import type { PageHelpBlock } from '../../data/page-help';
import { cn } from '../../styles/theme';

/** Splits on the inline markers while keeping them, so index order is stable. */
const INLINE = /(`[^`]+`|\*\*[^*]+\*\*|\[[^\]]+\]\([^)]+\))/g;
const LINK = /^\[([^\]]+)\]\(([^)]+)\)$/;

function inlineNode(part: string, key: number): ReactNode {
  if (part.startsWith('`') && part.endsWith('`')) {
    return (
      <code key={key} className="px-1 py-0.5 rounded bg-surface-hover font-mono text-xs">
        {part.slice(1, -1)}
      </code>
    );
  }
  if (part.startsWith('**') && part.endsWith('**')) {
    return (
      <strong key={key} className="font-semibold text-text-primary">
        {part.slice(2, -2)}
      </strong>
    );
  }
  const link = LINK.exec(part);
  if (link) {
    return (
      <a
        key={key}
        href={link[2]}
        target="_blank"
        rel="noreferrer"
        className="text-brand-accent underline"
      >
        {link[1]}
      </a>
    );
  }
  return part;
}

function richText(text: string): ReactNode[] {
  return text.split(INLINE).filter(Boolean).map(inlineNode);
}

interface PageHelpBodyProps {
  blocks: PageHelpBlock[];
}

export function PageHelpBody({ blocks }: PageHelpBodyProps): ReactElement {
  return (
    <div className="stack-default">
      {blocks.map((block, index) => {
        const key = `${block.kind}-${index}`;
        if (block.kind === 'paragraph') {
          return (
            <p key={key} className="text-sm text-text-muted leading-relaxed">
              {richText(block.text)}
            </p>
          );
        }
        if (block.kind === 'heading') {
          return (
            <h4 key={key} className="text-sm font-semibold text-text-primary">
              {richText(block.text)}
            </h4>
          );
        }
        if (block.kind === 'terms') {
          return (
            <dl key={key} className="stack-xs">
              {block.items.map((item) => (
                <div key={item.term} className="text-sm leading-relaxed">
                  <dt className="inline font-semibold text-text-primary">{item.term}</dt>{' '}
                  <dd className="inline text-text-muted">{richText(item.description)}</dd>
                </div>
              ))}
            </dl>
          );
        }
        return (
          <ul key={key} className={cn('stack-xs', 'list-disc pl-5')}>
            {block.items.map((item) => (
              <li key={item} className="text-sm text-text-muted leading-relaxed">
                {richText(item)}
              </li>
            ))}
          </ul>
        );
      })}
    </div>
  );
}

export default PageHelpBody;
