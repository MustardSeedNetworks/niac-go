import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { DeviceListPage } from '../../pages/DeviceListPage';
import { SmallText } from '../../ui/Typography';

/**
 * Step 2 — add/edit devices. Embeds the existing Device Library page
 * (`DeviceListPage`) unmodified: it already reads/writes the daemon's
 * currently-active config (which the wizard just started the simulation
 * against in step 1), so device CRUD here is the same code path the
 * standalone "Devices" nav item uses. Clicking a device to edit it
 * navigates to the existing `/device-config/:hostname` route — the same
 * app-wide behavior every other device list uses.
 */
export const DevicesStep: FC = () => {
  const { t } = useTranslation('pages');
  return (
    <div className="stack" data-testid="wizard-step-devices-content">
      <SmallText className="text-text-muted">{t('newSimWizard.devices.help')}</SmallText>
      <DeviceListPage />
    </div>
  );
};
