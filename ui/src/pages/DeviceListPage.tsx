import {
  AlertCircle,
  Filter,
  LayoutGrid,
  LayoutList,
  Plus,
  RefreshCw,
  Search,
  Server,
  Trash2,
  X,
} from "lucide-react";
import { type FC, useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { cloneDevice, deleteDevice, fetchConfigDevices } from "../api/client";
import type { Device, DeviceType } from "../api/types";
import {
  CloneDeviceModal,
  ConfirmDeleteModal,
  DeviceCardView,
  DeviceTableView,
} from "../components/device-list";
import { useApiResource } from "../hooks/useApiResource";
import {
  Button,
  Card,
  CardContent,
  ConfirmModal,
  DeviceCardGridSkeleton,
  DeviceTableSkeleton,
  H2,
  P,
  SmallText,
} from "../ui";
import { getErrorMessage } from "../utils";

export const DeviceListPage: FC = () => {
  const navigate = useNavigate();

  const {
    data: deviceList,
    loading,
    error,
    refetch,
  } = useApiResource(fetchConfigDevices, [], { intervalMs: 30000 });

  const [searchQuery, setSearchQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState<DeviceType | "all">("all");
  const [protocolFilter, setProtocolFilter] = useState<string>("all");
  const [viewMode, setViewMode] = useState<"cards" | "table">(() => {
    const stored = localStorage.getItem("niac-device-view-mode");
    return stored === "cards" || stored === "table" ? stored : "table";
  });
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);
  const [selectedDevices, setSelectedDevices] = useState<Set<string>>(new Set());
  const [showDeleteConfirm, setShowDeleteConfirm] = useState<string | null>(null);
  const [showCloneModal, setShowCloneModal] = useState<string | null>(null);
  const [showBulkDeleteConfirm, setShowBulkDeleteConfirm] = useState(false);

  // Persist view mode
  const handleViewModeChange = useCallback((mode: "cards" | "table") => {
    setViewMode(mode);
    localStorage.setItem("niac-device-view-mode", mode);
  }, []);

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
      if (d.snmp_agent) protos.add("SNMP");
      if (d.lldp?.enabled) protos.add("LLDP");
      if (d.cdp?.enabled) protos.add("CDP");
      if (d.stp?.enabled) protos.add("STP");
      if (d.dhcp) protos.add("DHCP");
      if (d.dns) protos.add("DNS");
      if (d.http?.enabled) protos.add("HTTP");
      if (d.ftp?.enabled) protos.add("FTP");
      if (d.netbios?.enabled) protos.add("NetBIOS");
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
      if (typeFilter !== "all" && device.type !== typeFilter) {
        return false;
      }

      // Protocol filter
      if (protocolFilter !== "all") {
        const hasProtocol =
          (protocolFilter === "SNMP" && device.snmp_agent) ||
          (protocolFilter === "LLDP" && device.lldp?.enabled) ||
          (protocolFilter === "CDP" && device.cdp?.enabled) ||
          (protocolFilter === "STP" && device.stp?.enabled) ||
          (protocolFilter === "DHCP" && device.dhcp) ||
          (protocolFilter === "DNS" && device.dns) ||
          (protocolFilter === "HTTP" && device.http?.enabled) ||
          (protocolFilter === "FTP" && device.ftp?.enabled) ||
          (protocolFilter === "NetBIOS" && device.netbios?.enabled);
        if (!hasProtocol) return false;
      }

      return true;
    });
  }, [devices, searchQuery, typeFilter, protocolFilter]);

  // Get protocols enabled for a device
  const getDeviceProtocols = useCallback((device: Device): string[] => {
    const protos: string[] = [];
    if (device.snmp_agent) protos.push("SNMP");
    if (device.lldp?.enabled) protos.push("LLDP");
    if (device.cdp?.enabled) protos.push("CDP");
    if (device.stp?.enabled) protos.push("STP");
    if (device.dhcp) protos.push("DHCP");
    if (device.dns) protos.push("DNS");
    if (device.http?.enabled) protos.push("HTTP");
    if (device.ftp?.enabled) protos.push("FTP");
    if (device.netbios?.enabled) protos.push("NetBIOS");
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
  const handleDelete = useCallback(
    async (hostname: string) => {
      try {
        await deleteDevice(hostname);
        setMessage({ type: "success", text: `Device "${hostname}" deleted successfully` });
        setShowDeleteConfirm(null);
        setSelectedDevices((prev) => {
          const next = new Set(prev);
          next.delete(hostname);
          return next;
        });
        refetch();
      } catch (err) {
        setMessage({ type: "error", text: getErrorMessage(err) });
      }
    },
    [refetch],
  );

  // Handle device clone
  const handleClone = useCallback(
    async (hostname: string, newHostname: string) => {
      try {
        await cloneDevice(hostname, { new_hostname: newHostname });
        setMessage({ type: "success", text: `Device cloned as "${newHostname}"` });
        setShowCloneModal(null);
        refetch();
      } catch (err) {
        setMessage({ type: "error", text: getErrorMessage(err) });
      }
    },
    [refetch],
  );

  // Handle bulk delete confirmation
  const handleBulkDeleteConfirm = useCallback(async () => {
    setShowBulkDeleteConfirm(false);

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
      setMessage({ type: "success", text: `${successCount} devices deleted successfully` });
    } else {
      setMessage({ type: "error", text: `Deleted ${successCount} devices, ${errorCount} failed` });
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
                onClick={() => navigate("/device-config/new")}
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
                  onClick={() => setSearchQuery("")}
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
                onChange={(e) => setTypeFilter(e.target.value as DeviceType | "all")}
                className="rounded-lg border border-white/10 bg-gray-950/60 py-2 px-3 text-sm text-white focus:border-violet-400 focus:outline-none"
              >
                <option value="all">All Types</option>
                {deviceTypes.map((type) => (
                  <option key={type} value={type}>
                    {type.replace("_", " ")}
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

            {/* View toggle */}
            <div className="flex items-center gap-1 p-1 rounded-lg bg-gray-950/60 border border-white/10">
              <button
                onClick={() => handleViewModeChange("table")}
                className={`p-2 rounded-md transition-colors ${
                  viewMode === "table"
                    ? "bg-violet-600 text-white"
                    : "text-gray-400 hover:text-white hover:bg-white/10"
                }`}
                title="Table view"
                aria-label="Table view"
              >
                <LayoutList className="h-4 w-4" />
              </button>
              <button
                onClick={() => handleViewModeChange("cards")}
                className={`p-2 rounded-md transition-colors ${
                  viewMode === "cards"
                    ? "bg-violet-600 text-white"
                    : "text-gray-400 hover:text-white hover:bg-white/10"
                }`}
                title="Card view"
                aria-label="Card view"
              >
                <LayoutGrid className="h-4 w-4" />
              </button>
            </div>
          </div>

          {/* Bulk actions */}
          {selectedDevices.size > 0 && (
            <div className="flex items-center gap-4 p-3 rounded-lg bg-violet-500/10 border border-violet-500/30">
              <span className="text-sm text-violet-200">
                {selectedDevices.size} device{selectedDevices.size !== 1 ? "s" : ""} selected
              </span>
              <Button
                variant="ghost"
                size="sm"
                leftIcon={<Trash2 className="h-4 w-4" />}
                onClick={() => setShowBulkDeleteConfirm(true)}
                className="text-red-400 hover:text-red-300 hover:bg-red-500/20"
              >
                Delete Selected
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setSelectedDevices(new Set())}>
                Clear Selection
              </Button>
            </div>
          )}

          {/* Status message */}
          {message && (
            <div
              className={`flex items-center gap-2 rounded-lg p-3 ${
                message.type === "success"
                  ? "border border-green-500/30 bg-green-500/10 text-green-300"
                  : "border border-red-500/30 bg-red-500/10 text-red-300"
              }`}
              role="alert"
            >
              {message.type === "error" && <AlertCircle className="h-4 w-4" />}
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
      {loading &&
        !deviceList &&
        (viewMode === "table" ? (
          <Card className="border-white/5 bg-gray-900/70">
            <CardContent className="p-0">
              {/* Table header skeleton */}
              <div className="flex items-center gap-4 border-b border-white/10 px-4 py-3 bg-gray-950/40">
                <div className="h-4 w-4 rounded bg-gray-700/50" />
                <div className="flex-1 grid grid-cols-12 gap-4 text-sm font-medium text-gray-400">
                  <div className="col-span-3">Hostname</div>
                  <div className="col-span-2">Type</div>
                  <div className="col-span-2">IP Address</div>
                  <div className="col-span-3">Protocols</div>
                  <div className="col-span-2 text-right">Actions</div>
                </div>
              </div>
              <DeviceTableSkeleton rows={8} />
            </CardContent>
          </Card>
        ) : (
          <DeviceCardGridSkeleton count={8} />
        ))}

      {/* Error state */}
      {error && (
        <Card className="border-red-500/30 bg-red-900/20">
          <CardContent className="space-y-3">
            <div className="flex items-start gap-3">
              <AlertCircle className="mt-1 h-5 w-5 text-red-400" />
              <div>
                <p className="font-semibold text-red-200">Failed to Load Devices</p>
                <SmallText className="text-red-300/90">{error.message}</SmallText>
                <Button variant="outline" size="sm" className="mt-3" onClick={() => refetch()}>
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
              onClick={() => navigate("/device-config/new")}
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
                setSearchQuery("");
                setTypeFilter("all");
                setProtocolFilter("all");
              }}
            >
              Clear Filters
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Device list - Table View */}
      {filteredDevices.length > 0 && viewMode === "table" && (
        <DeviceTableView
          devices={filteredDevices}
          selectedDevices={selectedDevices}
          onSelectDevice={toggleDeviceSelection}
          onSelectAll={handleSelectAll}
          onEdit={(hostname) => navigate(`/device-config/${encodeURIComponent(hostname)}`)}
          onClone={(hostname) => setShowCloneModal(hostname)}
          onDelete={(hostname) => setShowDeleteConfirm(hostname)}
          getDeviceProtocols={getDeviceProtocols}
        />
      )}

      {/* Device list - Card View */}
      {filteredDevices.length > 0 && viewMode === "cards" && (
        <DeviceCardView
          devices={filteredDevices}
          selectedDevices={selectedDevices}
          onSelectDevice={toggleDeviceSelection}
          onSelectAll={handleSelectAll}
          onEdit={(hostname) => navigate(`/device-config/${encodeURIComponent(hostname)}`)}
          onClone={(hostname) => setShowCloneModal(hostname)}
          onDelete={(hostname) => setShowDeleteConfirm(hostname)}
          getDeviceProtocols={getDeviceProtocols}
        />
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

      {/* Bulk delete confirmation modal */}
      <ConfirmModal
        isOpen={showBulkDeleteConfirm}
        onConfirm={handleBulkDeleteConfirm}
        onCancel={() => setShowBulkDeleteConfirm(false)}
        title="Delete Selected Devices"
        message={
          <>
            Are you sure you want to delete{" "}
            <strong>
              {selectedDevices.size} device{selectedDevices.size !== 1 ? "s" : ""}
            </strong>
            ? This action cannot be undone.
          </>
        }
        confirmLabel="Delete All"
        confirmTone="red"
      />
    </div>
  );
};
