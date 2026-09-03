import { AlertCircle, ArrowLeft, Check, RefreshCw, Save, Trash2 } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { DeviceType } from '../../api/types';
import { deviceTypeIcons } from '../../constants/device-types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';
import type { AuthoredDevice } from './generated/authored-device.generated';

export interface StatusMessage {
  type: 'success' | 'error';
  text: string;
}

export interface DeviceEditorHeaderProps {
  device: AuthoredDevice;
  isNewDevice: boolean;
  isDirty: boolean;
  saving: boolean;
  deleting: boolean;
  message: StatusMessage | null;
  showYamlPreview: boolean;
  onToggleYamlPreview: () => void;
  onDelete: () => void;
  onDiscard: () => void;
  onSave: () => void;
  onNavigateBack: () => void;
}

export const DeviceEditorHeader: FC<DeviceEditorHeaderProps> = ({
  device,
  isNewDevice,
  isDirty,
  saving,
  deleting,
  message,
  showYamlPreview,
  onToggleYamlPreview,
  onDelete,
  onDiscard,
  onSave,
  onNavigateBack,
}) => {
  const { t } = useTranslation('devices');
  const { t: tCommon } = useTranslation('common');
  const deviceType = device.type ?? 'unknown';
  const DeviceIcon = deviceTypeIcons[deviceType as DeviceType] ?? deviceTypeIcons.unknown;

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack-lg">
        <div className="flex flex-wrap items-center justify-between gap-comfortable">
          <div className="flex items-center gap-default">
            <button
              type="button"
              onClick={onNavigateBack}
              className="pad-xs text-text-muted hover:text-text-primary hover:bg-surface-hover rounded-lg transition-colors"
              title={t('editor.header.backTitle')}
            >
              <ArrowLeft className={iconSizes.lg} />
            </button>
            <DeviceIcon className={`${iconSizes.xl} text-brand-accent`} />
            <div>
              <H2>{isNewDevice ? t('editor.newDevice') : device.name || t('editor.editDevice')}</H2>
              <SmallText className="text-text-muted">
                {isNewDevice
                  ? t('editor.header.newDeviceSubtitle')
                  : t('editor.header.editDeviceSubtitle')}
              </SmallText>
            </div>
            {isDirty && (
              <Tag colorScheme="yellow" className="ml-inline">
                {t('editor.header.unsavedBadge')}
              </Tag>
            )}
          </div>
          <div className="flex gap-compact">
            <Button variant="outline" onClick={onToggleYamlPreview}>
              {showYamlPreview ? t('editor.header.hideYaml') : t('editor.header.showYaml')}
            </Button>
            {!isNewDevice && (
              <Button
                variant="outline"
                leftIcon={<Trash2 className={iconSizes.md} />}
                onClick={onDelete}
                className="text-status-error hover:text-status-error border-status-error/30 hover:border-status-error/50"
                disabled={deleting}
              >
                {tCommon('buttons.delete')}
              </Button>
            )}
            <Button variant="outline" onClick={onDiscard} disabled={!isDirty || saving}>
              {tCommon('buttons.discard')}
            </Button>
            <Button
              tone="violet"
              leftIcon={
                saving ? (
                  <RefreshCw className={`${iconSizes.md} animate-spin`} />
                ) : (
                  <Save className={iconSizes.md} />
                )
              }
              onClick={onSave}
              data-testid="device-editor-save"
              disabled={!isDirty || saving}
            >
              {saving
                ? tCommon('status.saving')
                : isNewDevice
                  ? tCommon('buttons.create')
                  : tCommon('buttons.save')}
            </Button>
          </div>
        </div>

        {/* Status message */}
        {message && (
          <div
            className={`flex items-center gap-compact rounded-lg pad-sm ${
              message.type === 'success'
                ? 'border border-status-success/30 bg-status-success/10 text-status-success'
                : 'border border-status-error/30 bg-status-error/10 text-status-error'
            }`}
            role="alert"
          >
            {message.type === 'success' ? (
              <Check className={iconSizes.md} />
            ) : (
              <AlertCircle className={iconSizes.md} />
            )}
            <span>{message.text}</span>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
