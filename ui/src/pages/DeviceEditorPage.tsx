import type { FC } from 'react';
import { useMemo } from 'react';
import {
  AdditionalIPsSection,
  BasicSettingsSection,
  buildYamlPreview,
  CdpSection,
  DeleteConfirmModal,
  DeviceEditorHeader,
  DeviceEditorStatusView,
  DhcpSection,
  DnsSection,
  FtpSection,
  HttpSection,
  LldpSection,
  NetBiosSection,
  SnmpSection,
  StpSection,
  TrafficSection,
  useDeviceEditor,
  YamlPreviewSection,
} from '../components/device-editor';

export const DeviceEditorPage: FC = () => {
  const {
    isNewDevice,
    device,
    isDirty,
    loading,
    error,
    refetch,
    walkFiles,
    visibleSections,
    saving,
    deleting,
    message,
    expandedSections,
    showYamlPreview,
    showDeleteConfirm,
    navigate,
    setShowYamlPreview,
    setShowDeleteConfirm,
    toggleSection,
    updateField,
    handleSave,
    handleDelete,
    handleDiscard,
  } = useDeviceEditor();

  // Generate YAML preview
  const yamlPreview = useMemo(() => buildYamlPreview(device), [device]);

  // Show loading or error state
  const statusView = (
    <DeviceEditorStatusView
      isNewDevice={isNewDevice}
      loading={loading}
      error={error}
      onRetry={refetch}
      onNavigateBack={() => navigate('/device-config')}
    />
  );

  if (!isNewDevice && (loading || error)) {
    return statusView;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <DeviceEditorHeader
        device={device}
        isNewDevice={isNewDevice}
        isDirty={isDirty}
        saving={saving}
        deleting={deleting}
        message={message}
        showYamlPreview={showYamlPreview}
        onToggleYamlPreview={() => setShowYamlPreview(!showYamlPreview)}
        onDelete={() => setShowDeleteConfirm(true)}
        onDiscard={handleDiscard}
        onSave={handleSave}
        onNavigateBack={() => navigate('/device-config')}
      />

      {/* YAML Preview */}
      {showYamlPreview && <YamlPreviewSection yamlContent={yamlPreview} />}

      {/* Basic Settings — always visible; type picker lives here so
          the rest of the form can react when it changes. */}
      {visibleSections.has('basic') && (
        <BasicSettingsSection
          device={device}
          isExpanded={expandedSections.has('basic')}
          onToggle={() => toggleSection('basic')}
          onUpdate={updateField}
        />
      )}

      {visibleSections.has('snmp') && (
        <SnmpSection
          device={device}
          isExpanded={expandedSections.has('snmp')}
          onToggle={() => toggleSection('snmp')}
          onUpdate={updateField}
          walkFiles={walkFiles}
        />
      )}

      {visibleSections.has('lldp') && (
        <LldpSection
          device={device}
          isExpanded={expandedSections.has('lldp')}
          onToggle={() => toggleSection('lldp')}
          onUpdate={updateField}
        />
      )}

      {visibleSections.has('cdp') && (
        <CdpSection
          device={device}
          isExpanded={expandedSections.has('cdp')}
          onToggle={() => toggleSection('cdp')}
          onUpdate={updateField}
        />
      )}

      {visibleSections.has('stp') && (
        <StpSection
          device={device}
          isExpanded={expandedSections.has('stp')}
          onToggle={() => toggleSection('stp')}
          onUpdate={updateField}
        />
      )}

      {visibleSections.has('ips') && (
        <AdditionalIPsSection
          device={device}
          isExpanded={expandedSections.has('ips')}
          onToggle={() => toggleSection('ips')}
          onUpdate={updateField}
        />
      )}

      {visibleSections.has('dhcp') && (
        <DhcpSection
          device={device}
          isExpanded={expandedSections.has('dhcp')}
          onToggle={() => toggleSection('dhcp')}
          onUpdate={updateField}
        />
      )}

      {visibleSections.has('dns') && (
        <DnsSection
          device={device}
          isExpanded={expandedSections.has('dns')}
          onToggle={() => toggleSection('dns')}
          onUpdate={updateField}
        />
      )}

      {visibleSections.has('http') && (
        <HttpSection
          device={device}
          isExpanded={expandedSections.has('http')}
          onToggle={() => toggleSection('http')}
          onUpdate={updateField}
        />
      )}

      {visibleSections.has('ftp') && (
        <FtpSection
          device={device}
          isExpanded={expandedSections.has('ftp')}
          onToggle={() => toggleSection('ftp')}
          onUpdate={updateField}
        />
      )}

      {visibleSections.has('netbios') && (
        <NetBiosSection
          device={device}
          isExpanded={expandedSections.has('netbios')}
          onToggle={() => toggleSection('netbios')}
          onUpdate={updateField}
        />
      )}

      {visibleSections.has('traffic') && (
        <TrafficSection
          device={device}
          isExpanded={expandedSections.has('traffic')}
          onToggle={() => toggleSection('traffic')}
          onUpdate={updateField}
        />
      )}

      {/* Delete Confirmation Modal */}
      {showDeleteConfirm && (
        <DeleteConfirmModal
          deviceHostname={device.hostname}
          deleting={deleting}
          onConfirm={handleDelete}
          onCancel={() => setShowDeleteConfirm(false)}
        />
      )}
    </div>
  );
};

export default DeviceEditorPage;
