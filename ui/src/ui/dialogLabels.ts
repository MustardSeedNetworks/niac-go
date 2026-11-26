import { useTranslation } from 'react-i18next';

/**
 * The translated labels every shared dialog needs and the shared set, which
 * has no i18n, cannot supply: the cancel button and the close button's name.
 */
export function useDialogLabels(): { cancelLabel: string; closeLabel: string } {
  const { t } = useTranslation('common');
  return { cancelLabel: t('buttons.cancel'), closeLabel: t('accessibility.closeDialog') };
}

/** The accessible name of a shared Modal's close button, which the shared set cannot translate. */
export function useCloseLabel(): string {
  const { t } = useTranslation('common');
  return t('accessibility.closeDialog');
}
