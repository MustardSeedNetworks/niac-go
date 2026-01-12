import { type FC } from 'react';
import { Folder, Plus, X } from 'lucide-react';
import { Button } from '../../ui';
import { CollapsibleSection, FormField } from '../form';
import type { FTPConfig, FTPUser } from '../../api/types';
import type { ProtocolSectionProps } from './types';
import { inputClassName } from './types';

export const FTPSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const updateFTP = (config: FTPConfig | undefined) => {
    onUpdate('ftp', config);
  };

  return (
    <CollapsibleSection
      title="FTP Server"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.ftp?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateFTP(enabled ? { enabled: true, system_type: 'UNIX Type: L8' } as FTPConfig : undefined);
      }}
    >
      {device.ftp?.enabled && (
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="Welcome Banner" helpText="FTP welcome message">
              <input
                type="text"
                value={device.ftp.welcome_banner || ''}
                onChange={(e) =>
                  updateFTP({ ...device.ftp!, welcome_banner: e.target.value })
                }
                placeholder="Welcome to FTP Server"
                className={inputClassName}
              />
            </FormField>

            <FormField label="System Type" helpText="SYST response">
              <input
                type="text"
                value={device.ftp.system_type || ''}
                onChange={(e) =>
                  updateFTP({ ...device.ftp!, system_type: e.target.value })
                }
                placeholder="UNIX Type: L8"
                className={inputClassName}
              />
            </FormField>

            <FormField label="Allow Anonymous" helpText="Allow anonymous FTP access">
              <label className="relative inline-flex items-center cursor-pointer mt-2">
                <input
                  type="checkbox"
                  checked={device.ftp.allow_anonymous ?? false}
                  onChange={(e) =>
                    updateFTP({ ...device.ftp!, allow_anonymous: e.target.checked })
                  }
                  className="sr-only peer"
                />
                <div className="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-violet-600 peer-focus:ring-2 peer-focus:ring-violet-500 transition-colors">
                  <div className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${device.ftp.allow_anonymous ? 'translate-x-4' : ''}`} />
                </div>
                <span className="ml-3 text-sm text-gray-300">{device.ftp.allow_anonymous ? 'Enabled' : 'Disabled'}</span>
              </label>
            </FormField>
          </div>

          {/* FTP Users */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-white flex items-center gap-2">
              <Folder className="h-4 w-4 text-violet-400" />
              FTP Users
            </h4>
            {(device.ftp.users || []).map((user, index) => (
              <div key={index} className="flex gap-2 items-center">
                <input
                  type="text"
                  value={user.username || ''}
                  onChange={(e) => {
                    const users = [...(device.ftp!.users || [])];
                    users[index] = { ...users[index], username: e.target.value };
                    updateFTP({ ...device.ftp!, users });
                  }}
                  placeholder="Username"
                  className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                />
                <input
                  type="text"
                  value={user.password || ''}
                  onChange={(e) => {
                    const users = [...(device.ftp!.users || [])];
                    users[index] = { ...users[index], password: e.target.value };
                    updateFTP({ ...device.ftp!, users });
                  }}
                  placeholder="Password"
                  className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                />
                <input
                  type="text"
                  value={user.home_dir || ''}
                  onChange={(e) => {
                    const users = [...(device.ftp!.users || [])];
                    users[index] = { ...users[index], home_dir: e.target.value };
                    updateFTP({ ...device.ftp!, users });
                  }}
                  placeholder="Home Directory"
                  className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                />
                <Button
                  variant="ghost"
                  tone="red"
                  size="sm"
                  onClick={() => {
                    const users = (device.ftp!.users || []).filter((_, i) => i !== index);
                    updateFTP({ ...device.ftp!, users });
                  }}
                >
                  <X className="h-4 w-4" />
                </Button>
              </div>
            ))}
            <Button
              variant="outline"
              size="sm"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => {
                const users = [...(device.ftp!.users || []), { username: '', password: '', home_dir: '/' } as FTPUser];
                updateFTP({ ...device.ftp!, users });
              }}
            >
              Add User
            </Button>
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
