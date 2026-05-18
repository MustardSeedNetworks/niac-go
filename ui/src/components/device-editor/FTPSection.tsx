import { Folder, Plus, X } from 'lucide-react';
import type { FC } from 'react';
import type { FTPConfig, FTPUser } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { CollapsibleSection, FormField } from '../form';
import type { ProtocolSectionProps } from './types';
import { inputClassName } from './types';

export const FtpSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const getFtpConfig = (): FTPConfig => ({
    enabled: true,
    systemType: 'UNIX Type: L8',
    users: [],
    ...(device.ftp ?? {}),
  });

  const updateFtp = (config: FTPConfig | undefined) => {
    onUpdate('ftp', config);
  };

  return (
    <CollapsibleSection
      title="FTP Server"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.ftp?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateFtp(enabled ? getFtpConfig() : undefined);
      }}
    >
      {device.ftp?.enabled && (
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="Welcome Banner" helpText="FTP welcome message">
              <input
                type="text"
                value={device.ftp.welcomeBanner || ''}
                onChange={(e) =>
                  updateFtp({
                    ...getFtpConfig(),
                    welcomeBanner: e.target.value,
                  })
                }
                placeholder="Welcome to FTP Server"
                className={inputClassName}
              />
            </FormField>

            <FormField label="System Type" helpText="SYST response">
              <input
                type="text"
                value={device.ftp.systemType || ''}
                onChange={(e) => updateFtp({ ...getFtpConfig(), systemType: e.target.value })}
                placeholder="UNIX Type: L8"
                className={inputClassName}
              />
            </FormField>

            <FormField label="Allow Anonymous" helpText="Allow anonymous FTP access">
              <label className="relative inline-flex items-center cursor-pointer mt-2">
                <input
                  type="checkbox"
                  checked={device.ftp.allowAnonymous ?? false}
                  onChange={(e) =>
                    updateFtp({
                      ...getFtpConfig(),
                      allowAnonymous: e.target.checked,
                    })
                  }
                  className="sr-only peer"
                />
                <div className="w-9 h-5 bg-bg-elevated rounded-full peer peer-checked:bg-brand-600 peer-focus:ring-2 peer-focus:ring-brand-500 transition-colors">
                  <div
                    className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${device.ftp.allowAnonymous ? 'translate-x-4' : ''}`}
                  />
                </div>
                <span className="ml-3 text-sm text-text-secondary">
                  {device.ftp.allowAnonymous ? 'Enabled' : 'Disabled'}
                </span>
              </label>
            </FormField>
          </div>

          {/* FTP Users */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-text-primary flex items-center gap-2">
              <Folder className={`${iconSizes.md} text-brand-400`} />
              FTP Users
            </h4>
            {(device.ftp.users || []).map((user: FTPUser, index: number) => (
              <div
                key={`${user.username || user.homeDir || 'user'}`}
                className="flex gap-2 items-center"
              >
                <input
                  type="text"
                  value={user.username || ''}
                  onChange={(e) => {
                    const users = [...(device.ftp?.users || [])];
                    users[index] = {
                      ...users[index],
                      username: e.target.value,
                    };
                    updateFtp({ ...getFtpConfig(), users });
                  }}
                  placeholder="Username"
                  className="flex-1 rounded-lg border border-white/10 bg-bg-base/60 p-2 text-sm text-text-primary placeholder-gray-500 focus:border-brand-400 focus:outline-none"
                />
                <input
                  type="text"
                  value={user.password || ''}
                  onChange={(e) => {
                    const users = [...(device.ftp?.users || [])];
                    users[index] = {
                      ...users[index],
                      password: e.target.value,
                    };
                    updateFtp({ ...getFtpConfig(), users });
                  }}
                  placeholder="Password"
                  className="flex-1 rounded-lg border border-white/10 bg-bg-base/60 p-2 text-sm text-text-primary placeholder-gray-500 focus:border-brand-400 focus:outline-none"
                />
                <input
                  type="text"
                  value={user.homeDir || ''}
                  onChange={(e) => {
                    const users = [...(device.ftp?.users || [])];
                    users[index] = { ...users[index], homeDir: e.target.value };
                    updateFtp({ ...getFtpConfig(), users });
                  }}
                  placeholder="Home Directory"
                  className="flex-1 rounded-lg border border-white/10 bg-bg-base/60 p-2 text-sm text-text-primary placeholder-gray-500 focus:border-brand-400 focus:outline-none font-mono"
                />
                <Button
                  variant="ghost"
                  tone="red"
                  size="sm"
                  onClick={() => {
                    const users = (device.ftp?.users || []).filter(
                      (_: FTPUser, i: number) => i !== index,
                    );
                    updateFtp({ ...getFtpConfig(), users });
                  }}
                >
                  <X className="h-4 w-4" />
                </Button>
              </div>
            ))}
            <Button
              variant="outline"
              size="sm"
              leftIcon={<Plus className={iconSizes.md} />}
              onClick={() => {
                const users = [
                  ...(device.ftp?.users || []),
                  { username: '', password: '', homeDir: '/' } as FTPUser,
                ];
                updateFtp({ ...getFtpConfig(), users });
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
