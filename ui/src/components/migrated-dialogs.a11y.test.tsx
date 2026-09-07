/**
 * migrated-dialogs.a11y.test.tsx — the five overlays that were hand-rolled (#1863).
 *
 * StreamView, ColoringRulesPanel, MergePreviewModal, TemplatePreviewModal and
 * CloneDeviceModal each drew their own `fixed inset-0` overlay beside `Modal`.
 * Two of them had no `role="dialog"` at all, two named themselves with a
 * literal `id="modal-title"` (a duplicate the moment a second dialog mounts),
 * and none of the five trapped focus or closed on Escape — so a keyboard user
 * who opened one was left tabbing through the page behind it.
 *
 * These assert the three properties the shared `Modal` gives them, per
 * dialog, because that is what a future hand-rolled overlay would silently
 * lose again.
 */
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import '../i18n';
import type { PcapPacket, Template } from '../api/types';
import { ColoringRulesPanel } from './ColoringRulesPanel';
import { MergePreviewModal } from './config/MergeControls';
import { CloneDeviceModal } from './device-list/CloneDeviceModal';
import { StreamView } from './StreamView';
import { TemplatePreviewModal } from './TemplatePreviewModal';

const packet: PcapPacket = {
  id: '1',
  number: 1,
  timestamp: new Date().toISOString(),
  protocol: 'TCP',
  sourceIp: '10.0.0.1',
  destIp: '10.0.0.2',
  sourcePort: 1234,
  destPort: 80,
  length: 64,
  info: 'packet',
  rawData: '68656c6c6f',
};

const template: Template = {
  name: 'hospital',
  description: 'Hospital pack',
  type: 'switch',
  deviceCount: 12,
};

/** [name the dialog announces, how to render it with this close handler] */
const dialogs: Array<[string, (onClose: () => void) => React.ReactElement]> = [
  [
    'Follow Stream',
    (onClose) => <StreamView packets={[packet]} clientEndpoint="10.0.0.1:1234" onClose={onClose} />,
  ],
  [
    'Coloring Rules',
    (onClose) => (
      <ColoringRulesPanel
        rules={[]}
        onRulesChange={() => undefined}
        onReset={() => undefined}
        onClose={onClose}
      />
    ),
  ],
  [
    'Merged Configuration Preview',
    (onClose) => (
      <MergePreviewModal content="devices: []" onClose={onClose} onExport={() => undefined} />
    ),
  ],
  [
    'hospital',
    (onClose) => (
      <TemplatePreviewModal
        template={template}
        content={null}
        loading={false}
        error={null}
        onClose={onClose}
        onUse={() => undefined}
        onCopy={() => undefined}
      />
    ),
  ],
  [
    'Clone Device',
    (onClose) => (
      <CloneDeviceModal hostname="core-sw-1" onClone={() => undefined} onCancel={onClose} />
    ),
  ],
];

// TemplatePreviewModal navigates on "edit a copy", so every dialog is rendered
// under a router rather than special-casing one of them.
function renderInRouter(element: React.ReactElement) {
  return render(<MemoryRouter>{element}</MemoryRouter>);
}

describe.each(dialogs)('%s', (name, renderDialog) => {
  it('is a dialog with an accessible name', () => {
    renderInRouter(renderDialog(() => undefined));
    expect(screen.getByRole('dialog', { name: new RegExp(name, 'i') })).toBeInTheDocument();
  });

  it('closes on Escape pressed inside it', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderInRouter(renderDialog(onClose));

    // The trap auto-focuses inside the dialog, which is where a real keypress
    // originates; pressing at the document would not exercise the same path.
    await user.keyboard('{Escape}');

    expect(onClose).toHaveBeenCalled();
  });

  it('keeps every focusable control inside the dialog', () => {
    renderInRouter(renderDialog(() => undefined));
    const dialog = screen.getByRole('dialog');

    for (const button of screen.getAllByRole('button')) {
      // The backdrop is Modal's own click-to-close control and sits outside
      // the trapped panel by design; everything else must be within it.
      if (button.getAttribute('aria-label') === 'Close dialog' && !dialog.contains(button)) {
        continue;
      }
      expect(dialog).toContainElement(button);
    }
  });
});

/**
 * A scrollable region needs keyboard access, but axe accepts focusable content
 * in place of a focusable container — so Modal gives the tab stop only to a
 * body that has none of its own. StreamView's stream dump is plain text;
 * ColoringRulesPanel's body is inputs and buttons.
 */
describe('scrollable dialog body', () => {
  function body(): HTMLElement {
    const element = screen.getByRole('dialog').querySelector('.overflow-y-auto');
    if (!(element instanceof HTMLElement)) {
      throw new Error('dialog has no scrollable body');
    }
    return element;
  }

  it('is focusable when its content is not', () => {
    renderInRouter(
      <StreamView packets={[packet]} clientEndpoint="10.0.0.1:1234" onClose={() => undefined} />,
    );
    expect(body()).toHaveAttribute('tabindex', '0');
  });

  it('is not an extra tab stop when its content is focusable', () => {
    // With no rules the panel renders an empty message and its inputs live in
    // the footer, so it needs one rule to have focusable body content at all.
    renderInRouter(
      <ColoringRulesPanel
        rules={[
          {
            id: 'rule-1',
            name: 'TCP',
            filter: 'tcp',
            foreground: '#ffffff',
            background: '#374151',
            enabled: true,
          },
        ]}
        onRulesChange={() => undefined}
        onReset={() => undefined}
        onClose={() => undefined}
      />,
    );
    expect(body()).not.toHaveAttribute('tabindex');
  });
});
