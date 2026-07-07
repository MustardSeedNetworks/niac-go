import { FileText, Plus, X } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { HTTPConfig, HTTPEndpoint } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { CollapsibleSection, FormField } from '../form';
import type { ProtocolSectionProps } from './types';
import { inputClassName } from './types';

export const HttpSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const { t } = useTranslation('devices');
  const getHttpConfig = (): HTTPConfig => ({
    enabled: true,
    serverName: 'NIAC/1.0',
    endpoints: [],
    ...(device.http ?? {}),
  });

  const updateHttp = (config: HTTPConfig | undefined) => {
    onUpdate('http', config);
  };

  return (
    <CollapsibleSection
      title={t('editor.sections.http.title')}
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.http?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateHttp(enabled ? getHttpConfig() : undefined);
      }}
    >
      {device.http?.enabled && (
        <div className="stack-xl">
          <div className="grid gap-comfortable md:grid-cols-2">
            <FormField
              label={t('editor.sections.http.serverNameLabel')}
              helpText={t('editor.sections.http.serverNameHelp')}
            >
              <input
                type="text"
                value={device.http.serverName || ''}
                onChange={(e) => updateHttp({ ...getHttpConfig(), serverName: e.target.value })}
                placeholder={t('editor.sections.http.serverNamePlaceholder')}
                className={inputClassName}
              />
            </FormField>
          </div>

          {/* Endpoints */}
          <div className="stack">
            <h4 className="label flex items-center gap-compact">
              <FileText className={`${iconSizes.md} text-brand-accent`} />
              {t('editor.sections.http.endpointsTitle')}
            </h4>
            {(device.http.endpoints || []).map((endpoint: HTTPEndpoint, index: number) => (
              <div
                key={`${endpoint.method || 'GET'}-${endpoint.path || endpoint.statusCode || 'endpoint'}`}
                className="rounded-lg border border-surface-border bg-bg-base/40 pad stack"
              >
                <div className="flex gap-compact items-center">
                  <select
                    value={endpoint.method || 'GET'}
                    onChange={(e) => {
                      const endpoints = [...(device.http?.endpoints || [])];
                      endpoints[index] = {
                        ...endpoints[index],
                        method: e.target.value as HTTPEndpoint['method'],
                      };
                      updateHttp({ ...getHttpConfig(), endpoints });
                    }}
                    className="w-24 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary focus:border-brand-accent focus:outline-none"
                  >
                    <option value="GET">GET</option>
                    <option value="POST">POST</option>
                    <option value="PUT">PUT</option>
                    <option value="DELETE">DELETE</option>
                  </select>
                  <input
                    type="text"
                    value={endpoint.path || ''}
                    onChange={(e) => {
                      const endpoints = [...(device.http?.endpoints || [])];
                      endpoints[index] = {
                        ...endpoints[index],
                        path: e.target.value,
                      };
                      updateHttp({ ...getHttpConfig(), endpoints });
                    }}
                    placeholder={t('editor.sections.http.pathPlaceholder')}
                    className="flex-1 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none font-mono"
                  />
                  <input
                    type="number"
                    value={endpoint.statusCode ?? 200}
                    onChange={(e) => {
                      const endpoints = [...(device.http?.endpoints || [])];
                      endpoints[index] = {
                        ...endpoints[index],
                        statusCode: Number.parseInt(e.target.value, 10),
                      };
                      updateHttp({ ...getHttpConfig(), endpoints });
                    }}
                    placeholder={t('editor.sections.http.statusPlaceholder')}
                    className="w-20 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                  />
                  <Button
                    variant="ghost"
                    tone="red"
                    size="sm"
                    onClick={() => {
                      const endpoints = (device.http?.endpoints || []).filter(
                        (_: HTTPEndpoint, i: number) => i !== index,
                      );
                      updateHttp({ ...getHttpConfig(), endpoints });
                    }}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
                <div className="flex gap-compact">
                  <input
                    type="text"
                    value={endpoint.contentType || ''}
                    onChange={(e) => {
                      const endpoints = [...(device.http?.endpoints || [])];
                      endpoints[index] = {
                        ...endpoints[index],
                        contentType: e.target.value,
                      };
                      updateHttp({ ...getHttpConfig(), endpoints });
                    }}
                    placeholder={t('editor.sections.http.contentTypePlaceholder')}
                    className="w-64 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                  />
                  <input
                    type="text"
                    value={endpoint.body || ''}
                    onChange={(e) => {
                      const endpoints = [...(device.http?.endpoints || [])];
                      endpoints[index] = {
                        ...endpoints[index],
                        body: e.target.value,
                      };
                      updateHttp({ ...getHttpConfig(), endpoints });
                    }}
                    placeholder={t('editor.sections.http.bodyPlaceholder')}
                    className="flex-1 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
                  />
                </div>
              </div>
            ))}
            <Button
              variant="outline"
              size="sm"
              leftIcon={<Plus className={iconSizes.md} />}
              onClick={() => {
                const endpoints = [
                  ...(device.http?.endpoints || []),
                  {
                    path: '/',
                    method: 'GET',
                    statusCode: 200,
                    contentType: 'text/html',
                  } as HTTPEndpoint,
                ];
                updateHttp({ ...getHttpConfig(), endpoints });
              }}
            >
              {t('editor.sections.http.addEndpointButton')}
            </Button>
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
