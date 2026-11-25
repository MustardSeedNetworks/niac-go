import { Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../shared/Button';
import { ActionButton } from '../../ui/ActionButton';

interface DeviceBulkActionsProps {
  selectedCount: number;
  onDeleteSelected: () => void;
  onClearSelection: () => void;
}

export function DeviceBulkActions({
  selectedCount,
  onDeleteSelected,
  onClearSelection,
}: DeviceBulkActionsProps) {
  const { t: tCommon } = useTranslation('common');
  const { t: tDevices } = useTranslation('devices');
  if (selectedCount === 0) {
    return null;
  }

  return (
    <div className="flex items-center gap-comfortable pad-sm rounded-lg bg-brand-primary/10 border border-brand-primary/30">
      <span className="text-sm text-brand-primary-strong">
        {tCommon('plurals.deviceCountSelected', { count: selectedCount })}
      </span>
      <ActionButton
        action="delete"
        variant="ghost"
        size="sm"
        icon={<Trash2 className="h-4 w-4" />}
        onClick={onDeleteSelected}
        className="text-status-error-strong hover:text-status-error-strong hover:bg-status-error/20"
      >
        {tDevices('list.bulkActions.deleteSelectedButton')}
      </ActionButton>
      <Button variant="ghost" size="sm" onClick={onClearSelection}>
        {tDevices('list.bulkActions.clearSelectionButton')}
      </Button>
    </div>
  );
}
