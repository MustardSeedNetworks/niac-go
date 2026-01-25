import type { FC } from 'react';
import { useMemo } from 'react';
import { CdpSection } from '../components/device-editor/CdpSection';
import { DeviceBasicSettings } from '../components/device-editor/DeviceBasicSettings';
import { DeviceDeleteModal } from '../components/device-editor/DeviceDeleteModal';
import { DeviceEditorHeader } from '../components/device-editor/DeviceEditorHeader';
import { DeviceIpAddresses } from '../components/device-editor/DeviceIpAddresses';
import { DeviceStatusView } from '../components/device-editor/DeviceStatusView';
import { DeviceYamlPreview } from '../components/device-editor/DeviceYamlPreview';
import { DhcpSection } from '../components/device-editor/DhcpSection';
import { DnsSection } from '../components/device-editor/DnsSection';
import { buildYamlPreview } from '../components/device-editor/deviceEditorUtils';
import { FtpSection } from '../components/device-editor/FtpSection';
import { HttpSection } from '../components/device-editor/HttpSection';
import { LldpSection } from '../components/device-editor/LldpSection';
import { NetBiosSection } from '../components/device-editor/NetBiosSection';
import { SnmpSection } from '../components/device-editor/SnmpSection';
import { StpSection } from '../components/device-editor/StpSection';
import { TrafficSection } from '../components/device-editor/TrafficSection';
import { useDeviceEditor } from '../components/device-editor/useDeviceEditor';

export const DeviceEditorPage: FC = () => {
  const {
    isNewDevice,
    navigate,
    device,
    saving,
    deleting,
    message,
    expandedSections,
    showYamlPreview,
    showDeleteConfirm,
    loading,
    error,
    walkFiles,
    isDirty,
    ipKeyCounter,
    ipKeysRef,
    toggleSection,
    updateField,
    handleSave,
    handleDelete,
    handleDiscard,
    refetch,
    setShowYamlPreview,
    setShowDeleteConfirm,
  } = useDeviceEditor();

  // Generate YAML preview
  const yamlPreview = useMemo(() => buildYamlPreview(device), [device]);

  // Check for loading/error states
  const statusView = (
    <DeviceStatusView
      isNewDevice={isNewDevice}
      loading={loading}
      error={error}
      refetch={refetch}
      navigate={navigate}
    />
  );

  if (statusView.props && ((!isNewDevice && loading) || (!isNewDevice && error))) {
    return statusView;
  }

  return (
    <div className="space-y-6">
      <DeviceEditorHeader
        device={device}
        isNewDevice={isNewDevice}
        isDirty={isDirty}
        saving={saving}
        deleting={deleting}
        message={message}
        showYamlPreview={showYamlPreview}
        onNavigateBack={() => navigate('/device-config')}
        onToggleYamlPreview={() => setShowYamlPreview(!showYamlPreview)}
        onShowDeleteConfirm={() => setShowDeleteConfirm(true)}
        onDiscard={handleDiscard}
        onSave={handleSave}
      />

      {showYamlPreview && <DeviceYamlPreview yamlContent={yamlPreview} />}

      <DeviceBasicSettings
        device={device}
        isExpanded={expandedSections.has('basic')}
        onToggle={() => toggleSection('basic')}
        onUpdate={updateField}
      />

      <SnmpSection
        device={device}
        isExpanded={expandedSections.has('snmp')}
        onToggle={() => toggleSection('snmp')}
        onUpdate={updateField}
        walkFiles={walkFiles}
      />

      <LldpSection
        device={device}
        isExpanded={expandedSections.has('lldp')}
        onToggle={() => toggleSection('lldp')}
        onUpdate={updateField}
      />

      <CdpSection
        device={device}
        isExpanded={expandedSections.has('cdp')}
        onToggle={() => toggleSection('cdp')}
        onUpdate={updateField}
      />

      <StpSection
        device={device}
        isExpanded={expandedSections.has('stp')}
        onToggle={() => toggleSection('stp')}
        onUpdate={updateField}
      />

      <DeviceIpAddresses
        device={device}
        isExpanded={expandedSections.has('ips')}
        onToggle={() => toggleSection('ips')}
        onUpdate={updateField}
        ipKeysRef={ipKeysRef}
        ipKeyCounterRef={ipKeyCounter}
      />

      <DhcpSection
        device={device}
        isExpanded={expandedSections.has('dhcp')}
        onToggle={() => toggleSection('dhcp')}
        onUpdate={updateField}
      />

      <DnsSection
        device={device}
        isExpanded={expandedSections.has('dns')}
        onToggle={() => toggleSection('dns')}
        onUpdate={updateField}
      />

      <HttpSection
        device={device}
        isExpanded={expandedSections.has('http')}
        onToggle={() => toggleSection('http')}
        onUpdate={updateField}
      />

      <FtpSection
        device={device}
        isExpanded={expandedSections.has('ftp')}
        onToggle={() => toggleSection('ftp')}
        onUpdate={updateField}
      />

      <NetBiosSection
        device={device}
        isExpanded={expandedSections.has('netbios')}
        onToggle={() => toggleSection('netbios')}
        onUpdate={updateField}
      />

      <TrafficSection
        device={device}
        isExpanded={expandedSections.has('traffic')}
        onToggle={() => toggleSection('traffic')}
        onUpdate={updateField}
      />

      {showDeleteConfirm && (
        <DeviceDeleteModal
          deviceHostname={device.hostname}
          deleting={deleting}
          onCancel={() => setShowDeleteConfirm(false)}
          onConfirm={handleDelete}
        />
      )}
    </div>
  );
};

export default DeviceEditorPage;
