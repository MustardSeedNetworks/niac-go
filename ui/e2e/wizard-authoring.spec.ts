import { expect, test } from '@playwright/test';

/**
 * P1b-3's acceptance: the wizard authors a complete network from empty.
 *
 * Driven against the daemon-served UI, so it exercises the real draft
 * lifecycle -- create, edit, revision-save -- rather than mocked routes. It
 * asserts the draft the wizard produced, which is the artefact the daemon
 * actually runs, not the shape of the form that produced it.
 */
test.describe('wizard authoring from empty', () => {
  test('gives every device an address and SNMP without leaving the wizard', async ({ page }) => {
    await page.goto('/new-simulation');

    // Capture the draft this run creates. Reading "the newest draft" from the
    // library made the assertion depend on which spec finished last.
    const created = page.waitForResponse(
      (response) =>
        response.url().includes('/api/v1/library/drafts') &&
        response.request().method() === 'POST' &&
        response.ok(),
    );

    // Step 1 - source and interface.
    const iface = page.getByTestId('wizard-interface-select');
    await expect(iface).toBeEnabled();
    await iface.selectOption({ index: 1 });
    await page.getByTestId('wizard-start-empty').click();
    await page.getByTestId('wizard-next-button').click();
    const draftName = ((await (await created).json()) as { name: string }).name;

    // Step 2 - author two devices through the composer, the surface an author
    // actually uses when starting from nothing.
    await expect(page.getByTestId('wizard-step-devices')).toHaveAttribute('data-status', 'active');
    for (const name of ['e2e-rtr-01', 'e2e-sw-01']) {
      await page.getByRole('button', { name: 'Add device' }).first().click();
      const dialog = page.getByRole('dialog');
      // useFocusTrap autofocuses the dialog's first focusable element (the
      // header's "Close modal" button, not the name field below) via
      // requestAnimationFrame once the dialog mounts. Filling the name field
      // before that rAF fires raced it: on a loaded runner the rAF could land
      // *after* fill() had already focused and typed into the name field,
      // stealing focus back to Close mid-fill and leaving the field empty --
      // which left the dialog's "Add device" button permanently disabled
      // (deviceValid requires a non-empty name) and the next click spun for
      // the full 30s test timeout instead of failing at its actual cause
      // (niac-go#1773). Waiting for that autofocus to land first sequences
      // around the race instead of fighting it after the fact.
      await expect(dialog.getByRole('button', { name: 'Close modal' })).toBeFocused();
      const nameField = dialog.getByLabel('Device name');
      await nameField.fill(name);
      await expect(nameField).toHaveValue(name);
      // Arm the wait before the click. Adding a device is a topology PATCH,
      // and the dialog closes only once it resolves: waiting on the dialog
      // alone made this a race against the network, which firefox lost on a
      // loaded runner and the retry then won.
      const added = page.waitForResponse(
        (response) =>
          response.url().includes('/topology') && response.request().method() === 'PATCH',
      );
      await dialog.getByRole('button', { name: 'Add device' }).click();
      await added;
      await expect(dialog).toBeHidden();
    }
    await page.getByTestId('wizard-next-button').click();

    // Step 3 - networks and addressing.
    await expect(page.getByTestId('wizard-step-networks')).toHaveAttribute('data-status', 'active');
    await page.getByTestId('networks-add').click();
    await page.locator('#network-name-0').fill('e2e-lan');
    await page.locator('#network-subnet-0').fill('10.77.0.0/24');
    await page.getByTestId('attachments-add').click();
    await page.getByTestId('addressing-assign-all').click();

    // Every device is addressed, and no two share an address.
    await expect(page.getByTestId('addressing-address-e2e-rtr-01')).toHaveText('10.77.0.1/24');
    await expect(page.getByTestId('addressing-address-e2e-sw-01')).toHaveText('10.77.0.2/24');
    await page.getByTestId('wizard-next-button').click();

    // Step 4 - protocols. SNMP is authored through the same generated section
    // the device editor renders.
    await expect(page.getByTestId('wizard-step-protocols')).toHaveAttribute(
      'data-status',
      'active',
    );
    const protocols = page.getByTestId('wizard-protocols-editor');
    await expect(protocols).toBeVisible();
    for (const device of ['e2e-rtr-01', 'e2e-sw-01']) {
      const card = protocols.locator('..').locator(`text=${device}`).first();
      await expect(card).toBeVisible();
    }
    await protocols.getByText('Snmp agent').first().click();
    await page.getByLabel('Community').first().fill('e2e_public');
    await page.getByTestId('wizard-next-button').click();

    // Step 5 - assert the saved draft, not the rendered preview. The draft is
    // the artefact the daemon actually runs, and the preview is a summary of
    // it rather than its YAML.
    await expect(page.getByTestId('wizard-step-review')).toHaveAttribute('data-status', 'active');

    const saved = await page.request.get(`/api/v1/library/drafts/${encodeURIComponent(draftName)}`);
    expect(saved.ok()).toBe(true);
    const { content } = (await saved.json()) as { content: string };

    // Authored from nothing: two devices, a network, an attachment, an
    // address each, and SNMP.
    expect(content).toContain('e2e-rtr-01');
    expect(content).toContain('e2e-sw-01');
    expect(content).toContain('e2e-lan');
    expect(content).toContain('10.77.0.0/24');
    expect(content).toContain('10.77.0.1/24');
    expect(content).toContain('10.77.0.2/24');
    expect(content).toContain('e2e_public');
  });
});
