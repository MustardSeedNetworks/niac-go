import { FileText, Plus, X } from "lucide-react";
import type { FC } from "react";
import type { HTTPConfig, HTTPEndpoint } from "../../api/types";
import { Button } from "../../ui";
import { CollapsibleSection, FormField } from "../form";
import type { ProtocolSectionProps } from "./types";
import { inputClassName } from "./types";

export const HTTPSection: FC<ProtocolSectionProps> = ({
  device,
  isExpanded,
  onToggle,
  onUpdate,
}) => {
  const updateHttp = (config: HTTPConfig | undefined) => {
    onUpdate("http", config);
  };

  return (
    <CollapsibleSection
      title="HTTP Server"
      isExpanded={isExpanded}
      onToggle={onToggle}
      enabled={device.http?.enabled ?? false}
      onEnableChange={(enabled) => {
        updateHttp(
          enabled ? ({ enabled: true, server_name: "NIAC/1.0" } as HTTPConfig) : undefined,
        );
      }}
    >
      {device.http?.enabled && (
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2">
            <FormField label="Server Name" helpText="HTTP Server header value">
              <input
                type="text"
                value={device.http.server_name || ""}
                onChange={(e) => updateHttp({ ...device.http!, server_name: e.target.value })}
                placeholder="Apache/2.4.41"
                className={inputClassName}
              />
            </FormField>
          </div>

          {/* Endpoints */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-white flex items-center gap-2">
              <FileText className="h-4 w-4 text-violet-400" />
              Endpoints
            </h4>
            {(device.http.endpoints || []).map((endpoint, index) => (
              <div
                key={index}
                className="rounded-lg border border-white/5 bg-gray-950/40 p-4 space-y-3"
              >
                <div className="flex gap-2 items-center">
                  <select
                    value={endpoint.method || "GET"}
                    onChange={(e) => {
                      const endpoints = [...(device.http?.endpoints || [])];
                      endpoints[index] = {
                        ...endpoints[index],
                        method: e.target.value as HTTPEndpoint["method"],
                      };
                      updateHttp({ ...device.http!, endpoints });
                    }}
                    className="w-24 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
                  >
                    <option value="GET">GET</option>
                    <option value="POST">POST</option>
                    <option value="PUT">PUT</option>
                    <option value="DELETE">DELETE</option>
                  </select>
                  <input
                    type="text"
                    value={endpoint.path || ""}
                    onChange={(e) => {
                      const endpoints = [...(device.http?.endpoints || [])];
                      endpoints[index] = { ...endpoints[index], path: e.target.value };
                      updateHttp({ ...device.http!, endpoints });
                    }}
                    placeholder="/api/status"
                    className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono"
                  />
                  <input
                    type="number"
                    value={endpoint.status_code ?? 200}
                    onChange={(e) => {
                      const endpoints = [...(device.http?.endpoints || [])];
                      endpoints[index] = {
                        ...endpoints[index],
                        status_code: parseInt(e.target.value, 10),
                      };
                      updateHttp({ ...device.http!, endpoints });
                    }}
                    placeholder="Status"
                    className="w-20 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                  />
                  <Button
                    variant="ghost"
                    tone="red"
                    size="sm"
                    onClick={() => {
                      const endpoints = (device.http?.endpoints || []).filter(
                        (_, i) => i !== index,
                      );
                      updateHttp({ ...device.http!, endpoints });
                    }}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={endpoint.content_type || ""}
                    onChange={(e) => {
                      const endpoints = [...(device.http?.endpoints || [])];
                      endpoints[index] = { ...endpoints[index], content_type: e.target.value };
                      updateHttp({ ...device.http!, endpoints });
                    }}
                    placeholder="Content-Type (e.g., application/json)"
                    className="w-64 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                  />
                  <input
                    type="text"
                    value={endpoint.body || ""}
                    onChange={(e) => {
                      const endpoints = [...(device.http?.endpoints || [])];
                      endpoints[index] = { ...endpoints[index], body: e.target.value };
                      updateHttp({ ...device.http!, endpoints });
                    }}
                    placeholder="Response body"
                    className="flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
                  />
                </div>
              </div>
            ))}
            <Button
              variant="outline"
              size="sm"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => {
                const endpoints = [
                  ...(device.http?.endpoints || []),
                  {
                    path: "/",
                    method: "GET",
                    status_code: 200,
                    content_type: "text/html",
                  } as HTTPEndpoint,
                ];
                updateHttp({ ...device.http!, endpoints });
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
