import { useTranslation } from 'react-i18next';
import type { DataTableLabels } from '../shared/DataTable';

interface TableLabelOptions<T> {
  /** Status line while loading, e.g. "Loading groups". */
  loading: string;
  /** Names a row's select checkbox. Needed only when the table has `selection`. */
  selectRow?: (row: T) => string;
}

/** The DataTable's fixed strings, translated once so no page restates them. */
export function useTableLabels<T>({
  loading,
  selectRow,
}: TableLabelOptions<T>): DataTableLabels<T> {
  const { t } = useTranslation('common');
  return {
    selectAll: t('dataTable.selectAll'),
    selectRow: selectRow ?? (() => ''),
    actions: t('dataTable.actionsColumn'),
    pagination: t('dataTable.pagination'),
    previousPage: t('dataTable.previousPage'),
    nextPage: t('dataTable.nextPage'),
    loading,
    pageStatus: (page, pageCount, total) => t('dataTable.pageStatus', { page, pageCount, total }),
  };
}
