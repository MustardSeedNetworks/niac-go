import { useTranslation } from 'react-i18next';
import { CategoryChip } from '../shared/CategoryChip';

const VISIBLE = 3;

/**
 * A device's protocols on one line: the first few as chips, the rest as a
 * count. One line is what keeps every row one height, which the virtualised
 * tables need, and what keeps 20 devices from filling a column.
 */
export function ProtocolChips({ protocols }: { protocols: readonly string[] }) {
  const { t } = useTranslation('common');
  if (protocols.length === 0) {
    return <span className="text-text-muted">{t('labels.none')}</span>;
  }
  const rest = protocols.length - VISIBLE;
  return (
    <div className="flex flex-nowrap items-center gap-1" title={protocols.join(', ')}>
      {protocols.slice(0, VISIBLE).map((protocol) => (
        <CategoryChip key={protocol} label={protocol} />
      ))}
      {rest > 0 ? <CategoryChip label={`+${rest}`} /> : null}
    </div>
  );
}
