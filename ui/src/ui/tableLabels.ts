import { useTranslation } from 'react-i18next';
import type { DataTableLabels } from '../shared/DataTable';

/**
 * The translated labels every shared DataTable needs and the shared set, which
 * has no i18n, cannot supply. `subject` names what the rows are ("devices"),
 * `selectRow` names one row's checkbox and is only read when rows are
 * selectable.
 */
export function useTableLabels<T>(
  subject: string,
  selectRow: (row: T) => string = () => '',
): DataTableLabels<T> {
  const { t } = useTranslation('common');
  return {
    selectAll: t('table.selectAll'),
    selectRow,
    actions: t('table.actions'),
    pagination: t('table.pagination'),
    previousPage: t('table.previousPage'),
    nextPage: t('table.nextPage'),
    loading: t('table.loading', { subject }),
    pageStatus: (page, pages, total) => t('table.pageStatus', { page, pages, total }),
  };
}
