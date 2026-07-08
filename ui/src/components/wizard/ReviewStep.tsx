import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { fetchConfig } from '../../api/client';
import { useApiResource } from '../../hooks/useApiResource';
import { SelectedNetworkPreview } from '../../pages/runtime/SelectedNetworkPreview';
import { SmallText } from '../../ui/Typography';

/**
 * Step 4 — review the config the wizard built. Reuses
 * `SelectedNetworkPreview` (Simulation page's "what's about to launch"
 * card) fed with the live daemon config content via its `content` prop,
 * instead of the usual template/userConfig/upload source fetch — the
 * wizard's config is already the daemon's active config by this point.
 */
export const ReviewStep: FC = () => {
  const { t } = useTranslation('pages');
  const { data, loading, error } = useApiResource(fetchConfig, []);

  if (loading && !data) {
    return <SmallText className="text-text-muted">{t('runtime.preview.loading')}</SmallText>;
  }

  if (error || !data) {
    return (
      <SmallText className="text-status-error" role="alert">
        {t('runtime.preview.loadError', { error: error?.message ?? '' })}
      </SmallText>
    );
  }

  return <SelectedNetworkPreview source="upload" name={data.filename} content={data.content} />;
};
