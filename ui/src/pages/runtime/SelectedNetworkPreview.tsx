import { Eye, Router, Server, Wifi } from 'lucide-react';
import { type FC, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { parse as parseYaml, YAMLParseError } from 'yaml';
import { fetchTemplateContent } from '../../api/client';
import { fetchLibraryNetworkContent } from '../../api/library-client';
import type { DeviceSummary } from '../../api/types';
import { DeviceTable } from '../../components/DeviceTable';
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
 *   content (raw YAML)   → skip the fetch entirely and preview it as-is
 *   source null          → render nothing
 *
 * The `content` prop lets the draft-first wizard preview saved content without
 * reading or changing the daemon's active configuration.
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
  /** Raw YAML to preview directly, bypassing the source-based fetch. */
  content?: string;
  view?: 'identity' | 'protocols';
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
  segments?: { devices?: ParsedDevice[] }[];
}

function typeIconFor(type: string) {
  const t = type.toLowerCase();
  if (t.includes('router') || t.includes('rtr') || t.includes('gateway')) return Router;
  if (t.includes('ap') || t.includes('wifi') || t.includes('wireless')) return Wifi;
  return Server;
}

function hasEnabledProperty(value: unknown): value is { enabled?: unknown } {
  return typeof value === 'object' && value !== null && 'enabled' in value;
}

function summariseDevice(d: ParsedDevice): DevicePreview {
  const ips: string[] = [];
  if (d.ip) ips.push(d.ip);
  if (Array.isArray(d.ips)) ips.push(...d.ips.filter((v): v is string => typeof v === 'string'));

  // YAML keys are snake_case; loop instead of indexing each one so
  // useLiteralKeys + useNamingConvention don't fight over the field names.
  const services: string[] = [];
  const explicitlyEnabledServices: [string, string][] = [
    ['snmpv3', 'SNMPv3'],
    ['lldp', 'LLDP'],
    ['cdp', 'CDP'],
    ['edp', 'EDP'],
    ['fdp', 'FDP'],
    ['stp', 'STP'],
    ['dhcpv6', 'DHCPv6'],
    ['http', 'HTTP'],
    ['ftp', 'FTP'],
    ['netbios', 'NetBIOS'],
    ['icmp', 'ICMP'],
    ['icmpv6', 'ICMPv6'],
    ['ssh', 'SSH'],
    ['syslog', 'Syslog'],
    ['iperf3', 'iPerf3'],
  ];
  for (const [key, label] of explicitlyEnabledServices) {
    const value = d[key];
    if (value === true || (hasEnabledProperty(value) && value.enabled === true)) {
      services.push(label);
    }
  }

  const presenceEnabledServices: [string, string][] = [
    ['dhcp', 'DHCP'],
    ['dns', 'DNS'],
    ['reflector', 'Reflector'],
  ];
  for (const [key, label] of presenceEnabledServices) {
    if (d[key] !== undefined && d[key] !== null && d[key] !== false) services.push(label);
  }

  const snmp = d.snmp_agent;
  if (snmp && typeof snmp === 'object' && (!hasEnabledProperty(snmp) || snmp.enabled !== false)) {
    services.push('SNMP');
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
  content,
  view = 'identity',
}) => {
  const { t } = useTranslation('pages');
  const [yamlText, setYamlText] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setError(null);

    if (content !== undefined) {
      setYamlText(content);
      return;
    }

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
  }, [source, name, uploadFile, content]);

  const { devices, parseError, parseErrorLine } = useMemo<{
    devices: DevicePreview[];
    parseError: string | null;
    parseErrorLine: number | null;
  }>(() => {
    if (!yamlText) return { devices: [], parseError: null, parseErrorLine: null };
    try {
      const parsed = parseYaml(yamlText) as ParsedYaml;
      const devices = [
        ...(Array.isArray(parsed?.devices) ? parsed.devices : []),
        ...(Array.isArray(parsed?.segments)
          ? parsed.segments.flatMap((segment) =>
              Array.isArray(segment.devices) ? segment.devices : [],
            )
          : []),
      ];
      if (devices.length === 0) {
        return { devices: [], parseError: null, parseErrorLine: null };
      }
      return {
        devices: devices.map(summariseDevice),
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

  if (!source && content === undefined) return null;

  const displayName = uploadFile?.name ?? name;

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack">
        <div className="flex items-baseline justify-between gap-default">
          <H2 className="flex items-center gap-compact text-lg">
            <Eye className={`${iconSizes.lg} text-brand-accent`} />
            {view === 'protocols'
              ? t('newSimWizard.protocols.title')
              : `${t('runtime.preview.selectedPrefix')} ${displayName}`}
          </H2>
          <SmallText className="text-text-muted">
            {view === 'protocols'
              ? t('newSimWizard.protocols.subtitle')
              : t('runtime.preview.previewOnly')}
          </SmallText>
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

        {view === 'identity' && devices.length > 0 && (
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

        {view === 'protocols' && devices.length > 0 && (
          <DeviceTable
            devices={devices.map<DeviceSummary>((device) => ({
              name: device.name,
              type: device.type,
              ips: device.ips,
              protocols: device.services,
              ...(device.mac ? { mac: device.mac } : {}),
            }))}
          />
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
