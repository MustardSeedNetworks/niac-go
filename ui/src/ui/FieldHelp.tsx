import { useTranslation } from 'react-i18next';
import { HelpIcon } from '../shared/HelpIcon';

/** The shared help button for a form field, named after the field it explains. */
export function FieldHelp({ field, text }: { field: string; text: string }) {
  const { t } = useTranslation('common');
  return <HelpIcon label={t('accessibility.fieldHelp', { field })} text={text} />;
}
