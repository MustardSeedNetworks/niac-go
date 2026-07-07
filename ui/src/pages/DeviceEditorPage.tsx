import type { FC } from 'react';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  AdditionalIPsSection,
  BasicSettingsSection,
  buildYamlPreview,
  CdpSection,
  DeviceEditorHeader,
  DeviceEditorStatusView,
  DhcpSection,
  DnsSection,
  FtpSection,
  HttpSection,
  InterfacesSection,
  LldpSection,
  NetBiosSection,
  SnmpSection,
  StpSection,
  TrafficSection,
  useDeviceEditor,
  YamlPreviewSection,
} from '../components/device-editor';
import { ConfirmModal } from '../ui/ConfirmModal';

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
    fieldErrors,
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
    requestNavigateBack,
    pendingLeavePath,
    confirmLeave,
    cancelLeave,
  } = useDeviceEditor();
  const { t } = useTranslation('devices');

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
    <div className="stack-xl">
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
        onNavigateBack={requestNavigateBack}
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
          errors={fieldErrors}
        />
      )}

      <InterfacesSection
        device={device}
        isExpanded={expandedSections.has('interfaces')}
        onToggle={() => toggleSection('interfaces')}
        onUpdate={updateField}
      />

      {visibleSections.has('snmp') && (
        <SnmpSection
          device={device}
          isExpanded={expandedSections.has('snmp')}
          onToggle={() => toggleSection('snmp')}
          onUpdate={updateField}
          walkFiles={walkFiles}
          isNewDevice={isNewDevice}
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
      <ConfirmModal
        isOpen={showDeleteConfirm}
        onConfirm={handleDelete}
        onCancel={() => setShowDeleteConfirm(false)}
        title={t('list.deleteConfirmTitle')}
        message={t('list.deleteConfirmMessage', { hostname: device.hostname })}
        confirmLabel={t('list.deleteConfirmLabel')}
        confirmTone="red"
        confirming={deleting}
        confirmingLabel={t('editor.deletingLabel')}
      />

      {/* Unsaved-changes navigation guard — covers the Back button and
          any in-app link clicked while the form is dirty. */}
      <ConfirmModal
        isOpen={pendingLeavePath !== null}
        onConfirm={confirmLeave}
        onCancel={cancelLeave}
        title={t('editor.unsavedNavigation.title')}
        message={t('editor.unsavedNavigation.message')}
        confirmLabel={t('editor.unsavedNavigation.confirmLabel')}
        cancelLabel={t('editor.unsavedNavigation.cancelLabel')}
      />
    </div>
  );
};

export default DeviceEditorPage;
