import { Folder, Plus, X } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
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
  const { t } = useTranslation('devices');
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
      title={t('editor.sections.ftp.title')}
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.ftp?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateFtp(enabled ? getFtpConfig() : undefined);
      }}
    >
      {device.ftp?.enabled && (
        <div className="stack-xl">
          <div className="grid gap-comfortable md:grid-cols-2">
            <FormField
              label={t('editor.sections.ftp.welcomeBannerLabel')}
              helpText={t('editor.sections.ftp.welcomeBannerHelp')}
            >
              <input
                type="text"
                value={device.ftp.welcomeBanner || ''}
                onChange={(e) =>
                  updateFtp({
                    ...getFtpConfig(),
                    welcomeBanner: e.target.value,
                  })
                }
                placeholder={t('editor.sections.ftp.welcomeBannerPlaceholder')}
                className={inputClassName}
              />
            </FormField>

            <FormField
              label={t('editor.sections.ftp.systemTypeLabel')}
              helpText={t('editor.sections.ftp.systemTypeHelp')}
            >
              <input
                type="text"
                value={device.ftp.systemType || ''}
                onChange={(e) => updateFtp({ ...getFtpConfig(), systemType: e.target.value })}
                placeholder={t('editor.sections.ftp.systemTypePlaceholder')}
                className={inputClassName}
              />
            </FormField>

            <FormField
              label={t('editor.sections.ftp.allowAnonymousLabel')}
              helpText={t('editor.sections.ftp.allowAnonymousHelp')}
            >
              <label className="relative inline-flex items-center cursor-pointer mt-inline">
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
                <div className="w-9 h-5 bg-bg-elevated rounded-full peer peer-checked:bg-brand-primary peer-focus:ring-2 peer-focus:ring-brand-primary transition-colors">
                  <div
                    className={`absolute top-0.5 left-0.5 w-4 h-4 bg-knob rounded-full transition-transform ${device.ftp.allowAnonymous ? 'translate-x-4' : ''}`}
                  />
                </div>
                <span className="ml-3 text-sm text-text-secondary">
                  {device.ftp.allowAnonymous
                    ? t('editor.sections.ftp.enabledState')
                    : t('editor.sections.ftp.disabledState')}
                </span>
              </label>
            </FormField>
          </div>

          {/* FTP Users */}
          <div className="stack">
            <h4 className="label flex items-center gap-compact">
              <Folder className={`${iconSizes.md} text-brand-accent`} />
              {t('editor.sections.ftp.usersTitle')}
            </h4>
            {(device.ftp.users || []).map((user: FTPUser, index: number) => (
              <div
                key={`${user.username || user.homeDir || 'user'}`}
                className="flex gap-compact items-center"
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
                  placeholder={t('editor.sections.ftp.usernamePlaceholder')}
                  className="flex-1 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                />
                <input
                  type="password"
                  autoComplete="new-password"
                  value={user.password || ''}
                  onChange={(e) => {
                    const users = [...(device.ftp?.users || [])];
                    users[index] = {
                      ...users[index],
                      password: e.target.value,
                    };
                    updateFtp({ ...getFtpConfig(), users });
                  }}
                  placeholder={t('editor.sections.ftp.passwordPlaceholder')}
                  className="flex-1 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                />
                <input
                  type="text"
                  value={user.homeDir || ''}
                  onChange={(e) => {
                    const users = [...(device.ftp?.users || [])];
                    users[index] = { ...users[index], homeDir: e.target.value };
                    updateFtp({ ...getFtpConfig(), users });
                  }}
                  placeholder={t('editor.sections.ftp.homeDirPlaceholder')}
                  className="flex-1 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none font-mono"
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
              {t('editor.sections.ftp.addUserButton')}
            </Button>
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
