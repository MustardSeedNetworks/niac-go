import { Eye, Router, Server, Wifi } from 'lucide-react';
import { type FC, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { parse as parseYaml, YAMLParseError } from 'yaml';
import { fetchTemplateContent } from '../../api/client';
import { fetchLibraryNetworkContent } from '../../api/library-client';
import { iconSizes } from '../../constants/sizes';
import type { ConfigSource } from '../../stores/ui-store';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';

/**
 * SelectedNetworkPreview shows what's about to launch when the user has
 * picked a network on the Simulation page but not yet hit Start. The
 * preview parses the selected config's YAML client-side and lists each
 * device with type, MAC, and IPs — so the user can verify their choice
 * before committing.
 *
 *   source 'template'    → fetchTemplateContent(name)
 *   source 'userConfig'  → fetchLibraryNetworkContent(name)
 *   source 'upload'      → read the File directly
 *   source null          → render nothing
 */
interface DevicePreview {
  name: string;
  type: string;
  mac?: string;
  ips: string[];
  services: string[];
}

interface SelectedNetworkPreviewProps {
  source: ConfigSource | null;
  name: string;
  uploadFile?: File | null;
}

// The YAML on disk uses snake_case (snmp_agent, netbios_status, …) which
// we'd otherwise have to enumerate as field names. Using an index-signature
// + safe lookups keeps the TS naming convention happy without lying about
// the input shape.
type ParsedDevice = {
  name?: string;
  type?: string;
  mac?: string;
  ip?: string;
  ips?: string[];
} & Record<string, unknown>;

interface ParsedYaml {
  devices?: ParsedDevice[];
}

function typeIconFor(type: string) {
  const t = type.toLowerCase();
  if (t.includes('router') || t.includes('rtr') || t.includes('gateway')) return Router;
  if (t.includes('ap') || t.includes('wifi') || t.includes('wireless')) return Wifi;
  return Server;
}

function summariseDevice(d: ParsedDevice): DevicePreview {
  const ips: string[] = [];
  if (d.ip) ips.push(d.ip);
  if (Array.isArray(d.ips)) ips.push(...d.ips.filter((v): v is string => typeof v === 'string'));

  // YAML keys are snake_case; loop instead of indexing each one so
  // useLiteralKeys + useNamingConvention don't fight over the field names.
  const services: string[] = [];
  const serviceKeys: [string, string][] = [
    ['snmp_agent', 'SNMP'],
    ['dhcp', 'DHCP'],
    ['dns', 'DNS'],
    ['http', 'HTTP'],
    ['netbios_status', 'NetBIOS'],
    ['icmp', 'ICMP'],
  ];
  for (const [key, label] of serviceKeys) {
    if (d[key]) services.push(label);
  }

  return {
    name: d.name ?? 'unnamed',
    type: d.type ?? 'host',
    mac: d.mac,
    ips,
    services,
  };
}

export const SelectedNetworkPreview: FC<SelectedNetworkPreviewProps> = ({
  source,
  name,
  uploadFile,
}) => {
  const { t } = useTranslation('pages');
  const [yamlText, setYamlText] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setError(null);

    if (!source || (!name && !uploadFile)) {
      setYamlText(null);
      return;
    }

    setLoading(true);
    const loader: Promise<string> =
      source === 'upload' && uploadFile
        ? uploadFile.text()
        : source === 'template'
          ? fetchTemplateContent(name).then((c) => c.content)
          : source === 'userConfig'
            ? fetchLibraryNetworkContent(name).then((c) => c.content)
            : Promise.resolve('');

    loader
      .then((text) => {
        if (!cancelled) setYamlText(text);
      })
      .catch((err: Error) => {
        if (!cancelled) setError(err.message);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [source, name, uploadFile]);

  const { devices, parseError, parseErrorLine } = useMemo<{
    devices: DevicePreview[];
    parseError: string | null;
    parseErrorLine: number | null;
  }>(() => {
    if (!yamlText) return { devices: [], parseError: null, parseErrorLine: null };
    try {
      const parsed = parseYaml(yamlText) as ParsedYaml;
      if (!parsed?.devices || !Array.isArray(parsed.devices)) {
        return { devices: [], parseError: null, parseErrorLine: null };
      }
      return {
        devices: parsed.devices.map(summariseDevice),
        parseError: null,
        parseErrorLine: null,
      };
    } catch (err) {
      // yaml's YAMLParseError carries a structured 1-based line/column
      // (linePos), so report it distinctly instead of a generic message —
      // mirrors the backend's parseYAMLError (internal/api/yaml_errors.go).
      const line = err instanceof YAMLParseError ? (err.linePos?.[0]?.line ?? null) : null;
      return { devices: [], parseError: (err as Error).message, parseErrorLine: line };
    }
  }, [yamlText]);

  if (!source) return null;

  const displayName = uploadFile?.name ?? name;

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack">
        <div className="flex items-baseline justify-between gap-default">
          <H2 className="flex items-center gap-compact text-lg">
            <Eye className={`${iconSizes.lg} text-brand-accent`} />
            {t('runtime.preview.selectedPrefix')} {displayName}
          </H2>
          <SmallText className="text-text-muted">{t('runtime.preview.previewOnly')}</SmallText>
        </div>

        {loading && (
          <SmallText className="text-text-muted">{t('runtime.preview.loading')}</SmallText>
        )}

        {error && (
          <SmallText className="text-status-error" role="alert">
            {t('runtime.preview.loadError', { error })}
          </SmallText>
        )}

        {!loading && !error && parseError && (
          <SmallText className="text-status-error" role="alert">
            {parseErrorLine !== null
              ? t('runtime.preview.parseErrorWithLine', { line: parseErrorLine, error: parseError })
              : t('runtime.preview.parseError', { error: parseError })}
          </SmallText>
        )}

        {!loading && !error && !parseError && devices.length === 0 && yamlText !== null && (
          <SmallText className="text-text-muted italic">{t('runtime.preview.noDevices')}</SmallText>
        )}

        {devices.length > 0 && (
          <ul className="grid grid-cols-1 gap-compact sm:grid-cols-2 lg:grid-cols-3">
            {devices.map((d) => {
              const Icon = typeIconFor(d.type);
              return (
                <li
                  key={d.name}
                  className="rounded-lg border border-surface-border bg-bg-base/40 px-3 py-row"
                >
                  <div className="flex items-center gap-compact">
                    <Icon className={`${iconSizes.sm} text-text-muted`} />
                    <span className="font-medium text-text-primary">{d.name}</span>
                    <Tag colorScheme="gray" className="text-[10px] capitalize">
                      {d.type}
                    </Tag>
                  </div>
                  {(d.mac || d.ips.length > 0) && (
                    <div className="mt-tight font-mono text-[11px] text-text-muted">
                      {d.mac && <span>{d.mac}</span>}
                      {d.mac && d.ips.length > 0 && <span> · </span>}
                      {d.ips.length > 0 && <span>{d.ips.join(', ')}</span>}
                    </div>
                  )}
                  {d.services.length > 0 && (
                    <div className="mt-tight flex flex-wrap gap-tight">
                      {d.services.map((svc) => (
                        <Tag key={svc} colorScheme="purple" className="text-[10px]">
                          {svc}
                        </Tag>
                      ))}
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        )}

        {devices.length > 0 && (
          <SmallText className="text-text-muted">
            {t('runtime.preview.deviceCount', { count: devices.length })}
          </SmallText>
        )}
      </CardContent>
    </Card>
  );
};
