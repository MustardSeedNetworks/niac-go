import { FileText, Plus, X } from 'lucide-react';
import type { FC } from 'react';
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
      title="HTTP Server"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.http?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateHttp(enabled ? getHttpConfig() : undefined);
      }}
    >
      {device.http?.enabled && (
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="Server Name" helpText="HTTP Server header value">
              <input
                type="text"
                value={device.http.serverName || ''}
                onChange={(e) => updateHttp({ ...getHttpConfig(), serverName: e.target.value })}
                placeholder="Apache/2.4.41"
                className={inputClassName}
              />
            </FormField>
          </div>

          {/* Endpoints */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-text-primary flex items-center gap-2">
              <FileText className={`${iconSizes.md} text-brand-400`} />
              Endpoints
            </h4>
            {(device.http.endpoints || []).map((endpoint: HTTPEndpoint, index: number) => (
              <div
                key={`${endpoint.method || 'GET'}-${endpoint.path || endpoint.statusCode || 'endpoint'}`}
                className="rounded-lg border border-white/5 bg-bg-base/40 p-4 space-y-3"
              >
                <div className="flex gap-2 items-center">
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
                    className="w-24 rounded-lg border border-white/10 bg-bg-base/60 p-2 text-sm text-text-primary focus:border-brand-400 focus:outline-none"
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
                    placeholder="/api/status"
                    className="flex-1 rounded-lg border border-white/10 bg-bg-base/60 p-2 text-sm text-text-primary placeholder-gray-500 focus:border-brand-400 focus:outline-none font-mono"
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
                    placeholder="Status"
                    className="w-20 rounded-lg border border-white/10 bg-bg-base/60 p-2 text-sm text-text-primary placeholder-gray-500 focus:border-brand-400 focus:outline-none"
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
                <div className="flex gap-2">
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
                    placeholder="Content-Type (e.g., application/json)"
                    className="w-64 rounded-lg border border-white/10 bg-bg-base/60 p-2 text-sm text-text-primary placeholder-gray-500 focus:border-brand-400 focus:outline-none"
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
                    placeholder="Response body"
                    className="flex-1 rounded-lg border border-white/10 bg-bg-base/60 p-2 text-sm text-text-primary placeholder-gray-500 focus:border-brand-400 focus:outline-none"
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
              Add Endpoint
            </Button>
          </div>
        </div>
      )}
    </CollapsibleSection>
  );
};
