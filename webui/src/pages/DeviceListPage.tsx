import { type FC, useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Search,
  Plus,
  X,
  Server,
  AlertCircle,
  RefreshCw,
  Trash2,
  Copy,
  Edit3,
  Filter,
  Router,
  Monitor,
  Wifi,
  Shield,
  HardDrive,
  Cpu,
} from 'lucide-react';
import {
  Card,
  CardContent,
  Button,
  H2,
  P,
  SmallText,
  Tag,
} from '../ui';
import { useApiResource } from '../hooks/useApiResource';
import {
  fetchConfigDevices,
  deleteDevice,
  cloneDevice,
} from '../api/client';
import type { Device, DeviceType } from '../api/types';

// Device type icons mapping
const deviceTypeIcons: Record<DeviceType, typeof Server> = {
  router: Router,
  switch: Server,
  access_point: Wifi,
  firewall: Shield,
  server: HardDrive,
  workstation: Monitor,
  iot: Cpu,
  unknown: Server,
};

// Device type colors for badges
const deviceTypeColors: Record<DeviceType, 'blue' | 'green' | 'purple' | 'yellow' | 'red' | 'gray'> = {
  router: 'blue',
  switch: 'green',
  access_point: 'purple',
  firewall: 'red',
  server: 'yellow',
  workstation: 'gray',
  iot: 'purple',
  unknown: 'gray',
};

export const DeviceListPage: FC = () => {
  const navigate = useNavigate();

  const {
    data: deviceList,
    loading,
    error,
    refetch,
  } = useApiResource(fetchConfigDevices, [], { intervalMs: 30000 });

  const [searchQuery, setSearchQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState<DeviceType | 'all'>('all');
  const [protocolFilter, setProtocolFilter] = useState<string>('all');
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [selectedDevices, setSelectedDevices] = useState<Set<string>>(new Set());
  const [showDeleteConfirm, setShowDeleteConfirm] = useState<string | null>(null);
  const [showCloneModal, setShowCloneModal] = useState<string | null>(null);

  const devices = deviceList?.devices ?? [];

  // Get unique device types for filter
  const deviceTypes = useMemo(() => {
    const types = new Set<DeviceType>();
    devices.forEach((d) => {
      if (d.type) types.add(d.type);
    });
    return Array.from(types);
  }, [devices]);

  // Get unique protocols for filter
  const protocols = useMemo(() => {
    const protos = new Set<string>();
    devices.forEach((d) => {
      if (d.snmp_agent) protos.add('SNMP');
      if (d.lldp?.enabled) protos.add('LLDP');
      if (d.cdp?.enabled) protos.add('CDP');
      if (d.stp?.enabled) protos.add('STP');
      if (d.dhcp) protos.add('DHCP');
      if (d.dns) protos.add('DNS');
      if (d.http?.enabled) protos.add('HTTP');
      if (d.ftp?.enabled) protos.add('FTP');
      if (d.netbios?.enabled) protos.add('NetBIOS');
    });
    return Array.from(protos).sort();
  }, [devices]);

  // Filter devices
  const filteredDevices = useMemo(() => {
    return devices.filter((device) => {
      // Search filter
      if (searchQuery.trim()) {
        const query = searchQuery.toLowerCase();
        const matchesSearch =
          device.hostname.toLowerCase().includes(query) ||
          device.mac.toLowerCase().includes(query) ||
          device.ip?.toLowerCase().includes(query) ||
          device.ips?.some((ip) => ip.toLowerCase().includes(query)) ||
          device.type?.toLowerCase().includes(query);
        if (!matchesSearch) return false;
      }

      // Type filter
      if (typeFilter !== 'all' && device.type !== typeFilter) {
        return false;
      }

      // Protocol filter
      if (protocolFilter !== 'all') {
        const hasProtocol =
          (protocolFilter === 'SNMP' && device.snmp_agent) ||
          (protocolFilter === 'LLDP' && device.lldp?.enabled) ||
          (protocolFilter === 'CDP' && device.cdp?.enabled) ||
          (protocolFilter === 'STP' && device.stp?.enabled) ||
          (protocolFilter === 'DHCP' && device.dhcp) ||
          (protocolFilter === 'DNS' && device.dns) ||
          (protocolFilter === 'HTTP' && device.http?.enabled) ||
          (protocolFilter === 'FTP' && device.ftp?.enabled) ||
          (protocolFilter === 'NetBIOS' && device.netbios?.enabled);
        if (!hasProtocol) return false;
      }

      return true;
    });
  }, [devices, searchQuery, typeFilter, protocolFilter]);

  // Get protocols enabled for a device
  const getDeviceProtocols = useCallback((device: Device): string[] => {
    const protos: string[] = [];
    if (device.snmp_agent) protos.push('SNMP');
    if (device.lldp?.enabled) protos.push('LLDP');
    if (device.cdp?.enabled) protos.push('CDP');
    if (device.stp?.enabled) protos.push('STP');
    if (device.dhcp) protos.push('DHCP');
    if (device.dns) protos.push('DNS');
    if (device.http?.enabled) protos.push('HTTP');
    if (device.ftp?.enabled) protos.push('FTP');
    if (device.netbios?.enabled) protos.push('NetBIOS');
    return protos;
  }, []);

  // Handle device selection
  const toggleDeviceSelection = useCallback((hostname: string) => {
    setSelectedDevices((prev) => {
      const next = new Set(prev);
      if (next.has(hostname)) {
        next.delete(hostname);
      } else {
        next.add(hostname);
      }
      return next;
    });
  }, []);

  // Handle select all
  const handleSelectAll = useCallback(() => {
    if (selectedDevices.size === filteredDevices.length) {
      setSelectedDevices(new Set());
    } else {
      setSelectedDevices(new Set(filteredDevices.map((d) => d.hostname)));
    }
  }, [selectedDevices.size, filteredDevices]);

  // Handle device deletion
  const handleDelete = useCallback(async (hostname: string) => {
    try {
      await deleteDevice(hostname);
      setMessage({ type: 'success', text: `Device "${hostname}" deleted successfully` });
      setShowDeleteConfirm(null);
      setSelectedDevices((prev) => {
        const next = new Set(prev);
        next.delete(hostname);
        return next;
      });
      refetch();
    } catch (err) {
      setMessage({ type: 'error', text: (err as Error).message });
    }
  }, [refetch]);

  // Handle device clone
  const handleClone = useCallback(async (hostname: string, newHostname: string) => {
    try {
      await cloneDevice(hostname, { new_hostname: newHostname });
      setMessage({ type: 'success', text: `Device cloned as "${newHostname}"` });
      setShowCloneModal(null);
      refetch();
    } catch (err) {
      setMessage({ type: 'error', text: (err as Error).message });
    }
  }, [refetch]);

  // Handle bulk delete
  const handleBulkDelete = useCallback(async () => {
    if (selectedDevices.size === 0) return;

    if (!window.confirm(`Delete ${selectedDevices.size} selected devices? This cannot be undone.`)) {
      return;
    }

    let successCount = 0;
    let errorCount = 0;

    for (const hostname of selectedDevices) {
      try {
        await deleteDevice(hostname);
        successCount++;
      } catch {
        errorCount++;
      }
    }

    setSelectedDevices(new Set());
    refetch();

    if (errorCount === 0) {
      setMessage({ type: 'success', text: `${successCount} devices deleted successfully` });
    } else {
      setMessage({ type: 'error', text: `Deleted ${successCount} devices, ${errorCount} failed` });
    }
  }, [selectedDevices, refetch]);

  return (
    <div className="space-y-6">
      {/* Header section */}
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <H2 className="mb-0 flex items-center gap-2">
                <Server className="h-5 w-5 text-violet-300" />
                Device Configuration
              </H2>
              <P className="text-gray-400 mt-1">
                Manage network device configurations for simulation.
                {devices.length > 0 && (
                  <span className="ml-2 text-violet-300">{devices.length} devices</span>
                )}
              </P>
            </div>
            <div className="flex gap-2">
              <Button
                variant="outline"
                leftIcon={<RefreshCw className="h-4 w-4" />}
                onClick={() => refetch()}
                disabled={loading}
              >
                Refresh
              </Button>
              <Button
                tone="violet"
                leftIcon={<Plus className="h-4 w-4" />}
                onClick={() => navigate('/device-config/new')}
              >
                Add Device
              </Button>
            </div>
          </div>

          {/* Search and filters */}
          <div className="flex flex-wrap gap-4">
            {/* Search */}
            <div className="relative flex-1 min-w-[250px]">
              <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-400" />
              <input
                type="text"
                placeholder="Search by hostname, MAC, or IP..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full rounded-lg border border-white/10 bg-gray-950/60 py-2.5 pl-10 pr-10 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery('')}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white"
                  aria-label="Clear search"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>

            {/* Type filter */}
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-gray-400" />
              <select
                value={typeFilter}
                onChange={(e) => setTypeFilter(e.target.value as DeviceType | 'all')}
                className="rounded-lg border border-white/10 bg-gray-950/60 py-2 px-3 text-sm text-white focus:border-violet-400 focus:outline-none"
              >
                <option value="all">All Types</option>
                {deviceTypes.map((type) => (
                  <option key={type} value={type}>
                    {type.replace('_', ' ')}
                  </option>
                ))}
              </select>
            </div>

            {/* Protocol filter */}
            <select
              value={protocolFilter}
              onChange={(e) => setProtocolFilter(e.target.value)}
              className="rounded-lg border border-white/10 bg-gray-950/60 py-2 px-3 text-sm text-white focus:border-violet-400 focus:outline-none"
            >
              <option value="all">All Protocols</option>
              {protocols.map((proto) => (
                <option key={proto} value={proto}>
                  {proto}
                </option>
              ))}
            </select>
          </div>

          {/* Bulk actions */}
          {selectedDevices.size > 0 && (
            <div className="flex items-center gap-4 p-3 rounded-lg bg-violet-500/10 border border-violet-500/30">
              <span className="text-sm text-violet-200">
                {selectedDevices.size} device{selectedDevices.size !== 1 ? 's' : ''} selected
              </span>
              <Button
                variant="ghost"
                size="sm"
                leftIcon={<Trash2 className="h-4 w-4" />}
                onClick={handleBulkDelete}
                className="text-red-400 hover:text-red-300 hover:bg-red-500/20"
              >
                Delete Selected
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setSelectedDevices(new Set())}
              >
                Clear Selection
              </Button>
            </div>
          )}

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
              {message.type === 'error' && <AlertCircle className="h-4 w-4" />}
              <span>{message.text}</span>
              <button
                onClick={() => setMessage(null)}
                className="ml-auto text-current hover:opacity-70"
                aria-label="Dismiss message"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Loading state */}
      {loading && !deviceList && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="flex items-center justify-center py-12">
            <div className="flex items-center gap-3 text-gray-400">
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-violet-500 border-t-transparent" />
              <span>Loading devices...</span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Error state */}
      {error && (
        <Card className="border-red-500/30 bg-red-900/20">
          <CardContent className="space-y-3">
            <div className="flex items-start gap-3">
              <AlertCircle className="mt-1 h-5 w-5 text-red-400" />
              <div>
                <p className="font-semibold text-red-200">Failed to Load Devices</p>
                <SmallText className="text-red-300/90">{error.message}</SmallText>
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-3"
                  onClick={() => refetch()}
                >
                  Retry
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Empty state */}
      {deviceList && devices.length === 0 && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="py-12 text-center">
            <Server className="mx-auto h-12 w-12 text-gray-600" />
            <H2 className="mt-4 mb-2">No Devices Configured</H2>
            <P className="text-gray-400">
              Add your first device to start configuring your network simulation.
            </P>
            <Button
              tone="violet"
              className="mt-4"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => navigate('/device-config/new')}
            >
              Add Device
            </Button>
          </CardContent>
        </Card>
      )}

      {/* No search results */}
      {devices.length > 0 && filteredDevices.length === 0 && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="py-12 text-center">
            <Search className="mx-auto h-12 w-12 text-gray-600" />
            <H2 className="mt-4 mb-2">No Matching Devices</H2>
            <P className="text-gray-400">
              No devices match your current filters. Try adjusting your search.
            </P>
            <Button
              variant="outline"
              className="mt-4"
              onClick={() => {
                setSearchQuery('');
                setTypeFilter('all');
                setProtocolFilter('all');
              }}
            >
              Clear Filters
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Device list */}
      {filteredDevices.length > 0 && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="p-0">
            {/* Table header */}
            <div className="flex items-center gap-4 border-b border-white/10 px-4 py-3 bg-gray-950/40">
              <input
                type="checkbox"
                checked={selectedDevices.size === filteredDevices.length && filteredDevices.length > 0}
                onChange={handleSelectAll}
                className="h-4 w-4 rounded border-gray-600 bg-gray-800 text-violet-500 focus:ring-violet-500"
              />
              <div className="flex-1 grid grid-cols-12 gap-4 text-sm font-medium text-gray-400">
                <div className="col-span-3">Hostname</div>
                <div className="col-span-2">Type</div>
                <div className="col-span-2">IP Address</div>
                <div className="col-span-3">Protocols</div>
                <div className="col-span-2 text-right">Actions</div>
              </div>
            </div>

            {/* Device rows */}
            <div className="divide-y divide-white/5">
              {filteredDevices.map((device) => {
                const DeviceIcon = deviceTypeIcons[device.type ?? 'unknown'];
                const typeColor = deviceTypeColors[device.type ?? 'unknown'];
                const deviceProtocols = getDeviceProtocols(device);

                return (
                  <div
                    key={device.hostname}
                    className={`flex items-center gap-4 px-4 py-3 hover:bg-white/5 transition-colors ${
                      selectedDevices.has(device.hostname) ? 'bg-violet-500/10' : ''
                    }`}
                  >
                    <input
                      type="checkbox"
                      checked={selectedDevices.has(device.hostname)}
                      onChange={() => toggleDeviceSelection(device.hostname)}
                      className="h-4 w-4 rounded border-gray-600 bg-gray-800 text-violet-500 focus:ring-violet-500"
                    />
                    <div className="flex-1 grid grid-cols-12 gap-4 items-center">
                      {/* Hostname */}
                      <div className="col-span-3 flex items-center gap-2">
                        <DeviceIcon className="h-4 w-4 text-gray-400" />
                        <button
                          onClick={() => navigate(`/device-config/${encodeURIComponent(device.hostname)}`)}
                          className="text-white hover:text-violet-300 font-medium truncate"
                        >
                          {device.hostname}
                        </button>
                      </div>

                      {/* Type */}
                      <div className="col-span-2">
                        <Tag colorScheme={typeColor} className="text-xs capitalize">
                          {device.type?.replace('_', ' ') || 'unknown'}
                        </Tag>
                      </div>

                      {/* IP */}
                      <div className="col-span-2">
                        <span className="text-gray-300 text-sm font-mono">
                          {device.ip || device.ips?.[0] || '—'}
                        </span>
                        {device.ips && device.ips.length > 1 && (
                          <span className="ml-1 text-gray-500 text-xs">
                            +{device.ips.length - 1}
                          </span>
                        )}
                      </div>

                      {/* Protocols */}
                      <div className="col-span-3 flex flex-wrap gap-1">
                        {deviceProtocols.length > 0 ? (
                          deviceProtocols.slice(0, 4).map((proto) => (
                            <Tag key={proto} colorScheme="gray" className="text-xs">
                              {proto}
                            </Tag>
                          ))
                        ) : (
                          <span className="text-gray-500 text-sm">No protocols</span>
                        )}
                        {deviceProtocols.length > 4 && (
                          <Tag colorScheme="gray" className="text-xs">
                            +{deviceProtocols.length - 4}
                          </Tag>
                        )}
                      </div>

                      {/* Actions */}
                      <div className="col-span-2 flex justify-end gap-1">
                        <button
                          onClick={() => navigate(`/device-config/${encodeURIComponent(device.hostname)}`)}
                          className="p-2 text-gray-400 hover:text-violet-300 hover:bg-white/5 rounded-lg transition-colors"
                          title="Edit device"
                        >
                          <Edit3 className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => setShowCloneModal(device.hostname)}
                          className="p-2 text-gray-400 hover:text-blue-300 hover:bg-white/5 rounded-lg transition-colors"
                          title="Clone device"
                        >
                          <Copy className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => setShowDeleteConfirm(device.hostname)}
                          className="p-2 text-gray-400 hover:text-red-400 hover:bg-white/5 rounded-lg transition-colors"
                          title="Delete device"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Delete confirmation modal */}
      {showDeleteConfirm && (
        <ConfirmDeleteModal
          hostname={showDeleteConfirm}
          onConfirm={() => handleDelete(showDeleteConfirm)}
          onCancel={() => setShowDeleteConfirm(null)}
        />
      )}

      {/* Clone device modal */}
      {showCloneModal && (
        <CloneDeviceModal
          hostname={showCloneModal}
          onClone={(newHostname) => handleClone(showCloneModal, newHostname)}
          onCancel={() => setShowCloneModal(null)}
        />
      )}
    </div>
  );
};

// Confirm Delete Modal
interface ConfirmDeleteModalProps {
  hostname: string;
  onConfirm: () => void;
  onCancel: () => void;
}

const ConfirmDeleteModal: FC<ConfirmDeleteModalProps> = ({ hostname, onConfirm, onCancel }) => {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
      onClick={onCancel}
      role="dialog"
      aria-modal="true"
    >
      <div
        className="mx-4 w-full max-w-md rounded-2xl border border-white/10 bg-gray-900/95 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-6 space-y-4">
          <div className="flex items-center gap-3 text-red-400">
            <Trash2 className="h-6 w-6" />
            <h2 className="text-lg font-semibold">Delete Device</h2>
          </div>
          <p className="text-gray-300">
            Are you sure you want to delete <strong>{hostname}</strong>? This action cannot be undone.
          </p>
          <div className="flex justify-end gap-3 pt-2">
            <Button variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button
              className="bg-red-600 hover:bg-red-700 text-white"
              onClick={onConfirm}
            >
              Delete
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
};

// Clone Device Modal
interface CloneDeviceModalProps {
  hostname: string;
  onClone: (newHostname: string) => void;
  onCancel: () => void;
}

const CloneDeviceModal: FC<CloneDeviceModalProps> = ({ hostname, onClone, onCancel }) => {
  const [newHostname, setNewHostname] = useState(`${hostname}-copy`);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newHostname.trim()) {
      onClone(newHostname.trim());
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
      onClick={onCancel}
      role="dialog"
      aria-modal="true"
    >
      <div
        className="mx-4 w-full max-w-md rounded-2xl border border-white/10 bg-gray-900/95 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div className="flex items-center gap-3 text-blue-400">
            <Copy className="h-6 w-6" />
            <h2 className="text-lg font-semibold">Clone Device</h2>
          </div>
          <p className="text-gray-300">
            Create a copy of <strong>{hostname}</strong> with a new name.
          </p>
          <div>
            <label htmlFor="new-hostname" className="block text-sm font-medium text-gray-300 mb-2">
              New Hostname
            </label>
            <input
              id="new-hostname"
              type="text"
              value={newHostname}
              onChange={(e) => setNewHostname(e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
              autoFocus
            />
          </div>
          <div className="flex justify-end gap-3 pt-2">
            <Button variant="outline" type="button" onClick={onCancel}>
              Cancel
            </Button>
            <Button tone="violet" type="submit" disabled={!newHostname.trim()}>
              Clone
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};
