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
    <div className="flex flex-wrap items-center gap-2" data-testid="device-bulk-actions">
      <span className="text-sm font-medium text-text-primary">
        {tCommon('plurals.deviceCountSelected', { count: selectedCount })}
      </span>
      <ActionButton
        action="delete"
        variant="destructive"
        size="sm"
        icon={<Trash2 className="h-4 w-4" />}
        onClick={onDeleteSelected}
      >
        {tDevices('list.bulkActions.deleteSelectedButton')}
      </ActionButton>
      <Button variant="ghost" size="sm" onClick={onClearSelection}>
        {tDevices('list.bulkActions.clearSelectionButton')}
      </Button>
    </div>
  );
}
