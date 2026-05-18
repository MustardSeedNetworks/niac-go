import { AlertCircle, ArrowLeft, Check, RefreshCw, Save, Trash2 } from 'lucide-react';
import type { FC } from 'react';
import type { Device, DeviceType } from '../../api/types';
import { deviceTypeIcons } from '../../constants/device-types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';

export interface StatusMessage {
  type: 'success' | 'error';
  text: string;
}

export interface DeviceEditorHeaderProps {
  device: Device;
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
  const deviceType = device.type ?? 'unknown';
  const DeviceIcon = deviceTypeIcons[deviceType as DeviceType] ?? deviceTypeIcons.unknown;

  return (
    <Card className="border-white/5 bg-bg-surface/70">
      <CardContent className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={onNavigateBack}
              className="p-2 text-text-muted hover:text-text-primary hover:bg-white/10 rounded-lg transition-colors"
              title="Back to device list"
            >
              <ArrowLeft className={iconSizes.lg} />
            </button>
            <DeviceIcon className={`${iconSizes.xl} text-brand-300`} />
            <div>
              <H2>{isNewDevice ? 'New Device' : device.hostname || 'Edit Device'}</H2>
              <SmallText className="text-text-muted">
                {isNewDevice ? 'Create a new network device' : 'Edit device configuration'}
              </SmallText>
            </div>
            {isDirty && (
              <Tag colorScheme="yellow" className="ml-2">
                Unsaved Changes
              </Tag>
            )}
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={onToggleYamlPreview}>
              {showYamlPreview ? 'Hide YAML' : 'Show YAML'}
            </Button>
            {!isNewDevice && (
              <Button
                variant="outline"
                leftIcon={<Trash2 className={iconSizes.md} />}
                onClick={onDelete}
                className="text-status-error hover:text-status-error border-status-error/30 hover:border-status-error/50"
                disabled={deleting}
              >
                Delete
              </Button>
            )}
            <Button variant="outline" onClick={onDiscard} disabled={!isDirty || saving}>
              Discard
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
              disabled={!isDirty || saving}
            >
              {saving ? 'Saving...' : isNewDevice ? 'Create' : 'Save'}
            </Button>
          </div>
        </div>

        {/* Status message */}
        {message && (
          <div
            className={`flex items-center gap-2 rounded-lg p-3 ${
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
