import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import type { ScenarioDeviceProfile } from '../../api/scenario-client';
import { Button } from '../../shared/Button';
import { FormField } from '../../shared/FormField';
import { Input } from '../../shared/Input';
import { Modal } from '../../shared/Modal';
import { Select } from '../../shared/Select';
import { ActionButton } from '../../ui/ActionButton';
import { useCloseLabel } from '../../ui/dialogLabels';

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

export function DeviceEditorModal({
  state,
  profiles,
  valid,
  busy,
  onChange,
  onSave,
}: DeviceEditorModalProps) {
  const closeLabel = useCloseLabel();
  const { t } = useTranslation('pages');
  const close = useCallback(() => onChange(null), [onChange]);
  const selectedProfile = state
    ? profiles.find((profile) => profile.role === state.role)
    : undefined;
  return (
    <Modal
      closeLabel={closeLabel}
      open={state !== null}
      onClose={close}
      title={t('newSimWizard.topology.addDevice')}
      size="lg"
      footer={
        state && (
          <>
            <Button variant="secondary" onClick={close}>
              {t('newSimWizard.topology.cancel')}
            </Button>
            <ActionButton
              variant="primary"
              action="edit"
              disabled={!valid || busy}
              onClick={onSave}
            >
              {t('newSimWizard.topology.addDevice')}
            </ActionButton>
          </>
        )
      }
    >
      {state && (
        <div className="stack-lg">
          <FormField label={t('newSimWizard.topology.deviceName')}>
            {(control) => (
              <Input
                {...control}
                value={state.name}
                onChange={(event) => onChange({ ...state, name: event.target.value })}
              />
            )}
          </FormField>
          <FormField label={t('newSimWizard.topology.profile')}>
            {(control) => (
              <Select
                {...control}
                value={state.role}
                options={profiles.map((profile) => ({
                  value: profile.role,
                  label: `${profile.role} · ${profile.model}`,
                }))}
                onValueChange={(role) => {
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
            )}
          </FormField>
          <div className="grid gap-default sm:grid-cols-2">
            <FormField label={t('newSimWizard.topology.interfaceCount')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  min={1}
                  max={4096}
                  disabled={Boolean(selectedProfile?.interfaces?.length)}
                  value={state.interfaceCount}
                  onChange={(event) => onChange({ ...state, interfaceCount: event.target.value })}
                />
              )}
            </FormField>
            <FormField label={t('newSimWizard.topology.speed')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  min={10}
                  max={400000}
                  disabled={Boolean(selectedProfile?.interfaces?.length)}
                  value={state.speed}
                  onChange={(event) => onChange({ ...state, speed: event.target.value })}
                />
              )}
            </FormField>
          </div>
        </div>
      )}
    </Modal>
  );
}

interface LinkEditorModalProps {
  state: LinkEditorState | null;
  valid: boolean;
  busy: boolean;
  interfaceOptions: (device: string, selected: string) => Array<{ value: string; label: string }>;
  onChange: (state: LinkEditorState | null) => void;
  onSave: () => void;
  onDisconnect: () => void;
}

export function LinkEditorModal({
  state,
  valid,
  busy,
  interfaceOptions,
  onChange,
  onSave,
  onDisconnect,
}: LinkEditorModalProps) {
  const closeLabel = useCloseLabel();
  const { t } = useTranslation('pages');
  const close = useCallback(() => onChange(null), [onChange]);
  return (
    <Modal
      closeLabel={closeLabel}
      open={state !== null}
      onClose={close}
      title={t('newSimWizard.topology.linkTitle')}
      size="lg"
      footer={
        state && (
          <>
            {state.existing && (
              <ActionButton
                action="edit"
                variant="destructive"
                disabled={busy}
                onClick={onDisconnect}
              >
                {t('newSimWizard.topology.disconnect')}
              </ActionButton>
            )}
            <Button variant="secondary" onClick={close}>
              {t('newSimWizard.topology.cancel')}
            </Button>
            <ActionButton
              variant="primary"
              action="edit"
              disabled={!valid || busy}
              onClick={onSave}
            >
              {t('newSimWizard.topology.saveLink')}
            </ActionButton>
          </>
        )
      }
    >
      {state && (
        <div className="stack-lg">
          <div className="grid gap-default sm:grid-cols-2">
            <FormField label={`${state.source} ${t('newSimWizard.topology.interface')}`}>
              {(control) => (
                <Select
                  {...control}
                  value={state.sourceInterface}
                  disabled={state.existing}
                  options={interfaceOptions(state.source, state.sourceInterface)}
                  placeholder={t('newSimWizard.topology.selectInterface')}
                  onValueChange={(sourceInterface) => onChange({ ...state, sourceInterface })}
                />
              )}
            </FormField>
            <FormField label={`${state.target} ${t('newSimWizard.topology.interface')}`}>
              {(control) => (
                <Select
                  {...control}
                  value={state.targetInterface}
                  disabled={state.existing}
                  options={interfaceOptions(state.target, state.targetInterface)}
                  placeholder={t('newSimWizard.topology.selectInterface')}
                  onValueChange={(targetInterface) => onChange({ ...state, targetInterface })}
                />
              )}
            </FormField>
          </div>
          <div className="grid gap-default sm:grid-cols-2">
            <FormField label={t('newSimWizard.topology.vlans')}>
              {(control) => (
                <Input
                  {...control}
                  value={state.vlans}
                  placeholder="200, 210"
                  onChange={(event) => onChange({ ...state, vlans: event.target.value })}
                />
              )}
            </FormField>
            <FormField label={t('newSimWizard.topology.nativeVlan')}>
              {(control) => (
                <Input
                  {...control}
                  type="number"
                  min={1}
                  max={4094}
                  value={state.nativeVlan}
                  onChange={(event) => onChange({ ...state, nativeVlan: event.target.value })}
                />
              )}
            </FormField>
          </div>
          <label className="flex min-h-11 items-center gap-default text-sm text-text-secondary">
            <input
              type="checkbox"
              checked={state.fdbOnly}
              onChange={(event) => onChange({ ...state, fdbOnly: event.target.checked })}
            />
            {t('newSimWizard.topology.fdbOnly')}
          </label>
        </div>
      )}
    </Modal>
  );
}
