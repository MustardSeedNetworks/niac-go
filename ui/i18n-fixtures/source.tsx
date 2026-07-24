import { useTranslation } from 'react-i18next';

export function ExtractionFixture({
  id,
  count,
  name,
}: {
  id: string;
  count: number;
  name: string;
}) {
  const { t } = useTranslation('common');
  const { t: tPages } = useTranslation('pages');
  const dynamicKey = `dynamic.${id}`;

  return (
    <>
      <span>{t('items', { count })}</span>
      <span>{t('greeting', { name })}</span>
      <span>{t('welcome', { context: 'formal' })}</span>
      <span>{t(dynamicKey)}</span>
      <span>{tPages('heading')}</span>
      <span>{tPages(dynamicKey)}</span>
    </>
  );
}
