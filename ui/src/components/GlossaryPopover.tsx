import { useTranslation } from 'react-i18next';
import type { GlossaryKey } from '../data/glossary';
import { InfoPopover } from '../ui/InfoPopover';

export function GlossaryPopover({ term }: { term: GlossaryKey }) {
  const { t } = useTranslation('help');
  const { t: tCommon } = useTranslation('common');
  const title = t(`glossary.${term}.term`);
  return (
    <InfoPopover label={tCommon('jargon.ariaLabel', { term: title })} title={title}>
      {t(`glossary.${term}.definition`)}
    </InfoPopover>
  );
}
