import { AlertCircle, ArrowLeft, Check, RefreshCw, Save, Trash2 } from 'lucide-react';
import type { FC } from 'react';
import type { Device } from '../../api/types';
import { deviceTypeIcons } from '../../constants/device-types';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';
import type { StatusMessage } from './deviceEditorUtils';

export interface DeviceEditorHeaderProps {
  device: Device;
  isNewDevice: boolean;
  isDirty: boolean;
  saving: boolean;
  deleting: boolean;
  message: StatusMessage | null;
  showYamlPreview: boolean;
  onNavigateBack: () => void;
  onToggleYamlPreview: () => void;
  onShowDeleteConfirm: () => void;
  onDiscard: () => void;
  onSave: () => void;
}

/**
 * Header section for the device editor with navigation, actions, and status
 */
export const DeviceEditorHeader: FC<DeviceEditorHeaderProps> = ({
  device,
  isNewDevice,
  isDirty,
  saving,
  deleting,
  message,
  showYamlPreview,
  onNavigateBack,
  onToggleYamlPreview,
  onShowDeleteConfirm,
  onDiscard,
  onSave,
}) => {
  const DeviceIcon = deviceTypeIcons[device.type ?? 'unknown'];

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={onNavigateBack}
              className="p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
              title="Back to device list"
            >
              <ArrowLeft className="h-5 w-5" />
            </button>
            <DeviceIcon className="h-6 w-6 text-violet-300" />
            <div>
              <H2 className="mb-0">
                {isNewDevice ? 'New Device' : device.hostname || 'Edit Device'}
              </H2>
              <SmallText className="text-gray-400">
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
                leftIcon={<Trash2 className="h-4 w-4" />}
                onClick={onShowDeleteConfirm}
                className="text-red-400 hover:text-red-300 border-red-400/30 hover:border-red-400/50"
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
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  <Save className="h-4 w-4" />
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
                ? 'border border-green-500/30 bg-green-500/10 text-green-300'
                : 'border border-red-500/30 bg-red-500/10 text-red-300'
            }`}
            role="alert"
          >
            {message.type === 'success' ? (
              <Check className="h-4 w-4" />
            ) : (
              <AlertCircle className="h-4 w-4" />
            )}
            <span>{message.text}</span>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
