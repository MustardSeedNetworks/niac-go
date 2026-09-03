import type { FC } from 'react';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  AdditionalIPsSection,
  BasicSettingsSection,
  DeviceEditorHeader,
  DeviceEditorStatusView,
  SchemaSectionBody,
  SynthesizeWalkControl,
  useDeviceEditor,
  YamlPreviewSection,
} from '../components/device-editor';
import type { AuthoredValue } from '../components/device-editor/generated/authored-device.generated';
import { CollapsibleSection } from '../components/form/CollapsibleSection';
import { ConfirmModal } from '../ui/ConfirmModal';

export const DeviceEditorPage: FC = () => {
  const {
    hostname,
    isNewDevice,
    device,
    isDirty,
    yaml,
    loading,
    error,
    refetch,
    walkFiles,
    sections,
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

  // The one open string field the daemon resolves against a library. Offered
  // as suggestions rather than a closed list, because a config may reference a
  // walk that has since been renamed and the author must still see it.
  const suggestions = useMemo(
    () => ({ 'snmp_agent.walk_file': (walkFiles ?? []).map((file) => file.name) }),
    [walkFiles],
  );

  if (!isNewDevice && (loading || error)) {
    return (
      <DeviceEditorStatusView
        isNewDevice={isNewDevice}
        loading={loading}
        error={error}
        onRetry={refetch}
        onNavigateBack={() => navigate('/device-config')}
      />
    );
  }

  return (
    <div className="stack-xl">
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

      {showYamlPreview && <YamlPreviewSection yamlContent={yaml} />}

      <BasicSettingsSection
        device={device}
        isNewDevice={isNewDevice}
        isExpanded={expandedSections.has('basic')}
        onToggle={() => toggleSection('basic')}
        onUpdate={updateField}
        errors={fieldErrors}
      />

      <AdditionalIPsSection
        device={device}
        isExpanded={expandedSections.has('ips')}
        onToggle={() => toggleSection('ips')}
        onUpdate={updateField}
        errors={fieldErrors}
      />

      {/* Every section the schema declares, in relevance order. None is
          hidden: a section the form does not render is a field the author
          cannot reach, and the authoring-parity gate would still count it
          bound. */}
      {sections.map((section) => (
        <CollapsibleSection
          key={section.key}
          id={`${section.key}-section`}
          title={t(`editor.sections.${section.key}.title`, { defaultValue: section.title })}
          isExpanded={expandedSections.has(section.key)}
          onToggle={() => toggleSection(section.key)}
        >
          <div className="stack-lg">
            <SchemaSectionBody
              section={section}
              value={device[section.key as keyof typeof device] as AuthoredValue}
              onChange={(next) => updateField(section.key as keyof typeof device, next)}
              suggestions={suggestions}
            />
            {section.key === 'snmp_agent' && (
              <SynthesizeWalkControl
                hostname={device.name ?? hostname ?? ''}
                disabled={isNewDevice}
                onSynthesized={(walkPath) =>
                  updateField('snmp_agent', {
                    ...(typeof device.snmp_agent === 'object' ? device.snmp_agent : {}),
                    walk_file: walkPath,
                  })
                }
              />
            )}
          </div>
        </CollapsibleSection>
      ))}

      <ConfirmModal
        isOpen={showDeleteConfirm}
        onConfirm={handleDelete}
        onCancel={() => setShowDeleteConfirm(false)}
        title={t('list.deleteConfirmTitle')}
        message={t('list.deleteConfirmMessage', { hostname: device.name ?? '' })}
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
