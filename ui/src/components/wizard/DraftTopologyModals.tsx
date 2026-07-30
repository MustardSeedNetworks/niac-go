import { type FC, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import type { ScenarioDeviceProfile } from '../../api/scenario-client';
import { Button } from '../../ui/Button';
import { Input, Select } from '../../ui/Input';
import { Modal, ModalBody, ModalFooter } from '../../ui/Modal';

export interface LinkEditorState {
  source: string;
  target: string;
  sourceInterface: string;
  targetInterface: string;
  vlans: string;
  nativeVlan: string;
  fdbOnly: boolean;
  existing: boolean;
}

export interface DeviceEditorState {
  name: string;
  role: string;
  interfaceCount: string;
  speed: string;
}

interface DeviceEditorModalProps {
  state: DeviceEditorState | null;
  profiles: ScenarioDeviceProfile[];
  valid: boolean;
  busy: boolean;
  onChange: (state: DeviceEditorState | null) => void;
  onSave: () => void;
}

export const DeviceEditorModal: FC<DeviceEditorModalProps> = ({
  state,
  profiles,
  valid,
  busy,
  onChange,
  onSave,
}) => {
  const { t } = useTranslation('pages');
  const close = useCallback(() => onChange(null), [onChange]);
  const selectedProfile = state
    ? profiles.find((profile) => profile.role === state.role)
    : undefined;
  return (
    <Modal
      isOpen={state !== null}
      onClose={close}
      title={t('newSimWizard.topology.addDevice')}
      size="lg"
    >
      {state && (
        <>
          <ModalBody>
            <Input
              label={t('newSimWizard.topology.deviceName')}
              value={state.name}
              onChange={(event) => onChange({ ...state, name: event.target.value })}
            />
            <Select
              label={t('newSimWizard.topology.profile')}
              value={state.role}
              options={profiles.map((profile) => ({
                value: profile.role,
                label: `${profile.role} · ${profile.model}`,
              }))}
              onChange={(role) => {
                const profile = profiles.find((item) => item.role === role);
                onChange({
                  ...state,
                  role,
                  interfaceCount: profile?.interfaceCount
                    ? String(profile.interfaceCount)
                    : state.interfaceCount,
                });
              }}
            />
            <div className="grid gap-default sm:grid-cols-2">
              <Input
                label={t('newSimWizard.topology.interfaceCount')}
                type="number"
                min={1}
                max={4096}
                disabled={Boolean(selectedProfile?.interfaces?.length)}
                value={state.interfaceCount}
                onChange={(event) => onChange({ ...state, interfaceCount: event.target.value })}
              />
              <Input
                label={t('newSimWizard.topology.speed')}
                type="number"
                min={10}
                max={400000}
                disabled={Boolean(selectedProfile?.interfaces?.length)}
                value={state.speed}
                onChange={(event) => onChange({ ...state, speed: event.target.value })}
              />
            </div>
          </ModalBody>
          <ModalFooter>
            <Button variant="outline" onClick={close}>
              {t('newSimWizard.topology.cancel')}
            </Button>
            <Button tone="violet" disabled={!valid || busy} onClick={onSave}>
              {t('newSimWizard.topology.addDevice')}
            </Button>
          </ModalFooter>
        </>
      )}
    </Modal>
  );
};

interface LinkEditorModalProps {
  state: LinkEditorState | null;
  valid: boolean;
  busy: boolean;
  interfaceOptions: (device: string, selected: string) => Array<{ value: string; label: string }>;
  onChange: (state: LinkEditorState | null) => void;
  onSave: () => void;
  onDisconnect: () => void;
}

export const LinkEditorModal: FC<LinkEditorModalProps> = ({
  state,
  valid,
  busy,
  interfaceOptions,
  onChange,
  onSave,
  onDisconnect,
}) => {
  const { t } = useTranslation('pages');
  const close = useCallback(() => onChange(null), [onChange]);
  return (
    <Modal
      isOpen={state !== null}
      onClose={close}
      title={t('newSimWizard.topology.linkTitle')}
      size="lg"
    >
      {state && (
        <>
          <ModalBody>
            <div className="grid gap-default sm:grid-cols-2">
              <Select
                label={`${state.source} ${t('newSimWizard.topology.interface')}`}
                value={state.sourceInterface}
                disabled={state.existing}
                options={interfaceOptions(state.source, state.sourceInterface)}
                placeholder={t('newSimWizard.topology.selectInterface')}
                onChange={(sourceInterface) => onChange({ ...state, sourceInterface })}
              />
              <Select
                label={`${state.target} ${t('newSimWizard.topology.interface')}`}
                value={state.targetInterface}
                disabled={state.existing}
                options={interfaceOptions(state.target, state.targetInterface)}
                placeholder={t('newSimWizard.topology.selectInterface')}
                onChange={(targetInterface) => onChange({ ...state, targetInterface })}
              />
            </div>
            <div className="grid gap-default sm:grid-cols-2">
              <Input
                label={t('newSimWizard.topology.vlans')}
                value={state.vlans}
                placeholder="200, 210"
                onChange={(event) => onChange({ ...state, vlans: event.target.value })}
              />
              <Input
                label={t('newSimWizard.topology.nativeVlan')}
                type="number"
                min={1}
                max={4094}
                value={state.nativeVlan}
                onChange={(event) => onChange({ ...state, nativeVlan: event.target.value })}
              />
            </div>
            <label className="flex min-h-11 items-center gap-default text-sm text-text-secondary">
              <input
                type="checkbox"
                checked={state.fdbOnly}
                onChange={(event) => onChange({ ...state, fdbOnly: event.target.checked })}
              />
              {t('newSimWizard.topology.fdbOnly')}
            </label>
          </ModalBody>
          <ModalFooter>
            {state.existing && (
              <Button tone="red" disabled={busy} onClick={onDisconnect}>
                {t('newSimWizard.topology.disconnect')}
              </Button>
            )}
            <Button variant="outline" onClick={close}>
              {t('newSimWizard.topology.cancel')}
            </Button>
            <Button tone="violet" disabled={!valid || busy} onClick={onSave}>
              {t('newSimWizard.topology.saveLink')}
            </Button>
          </ModalFooter>
        </>
      )}
    </Modal>
  );
};
