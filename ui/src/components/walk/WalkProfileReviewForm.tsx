import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import type { WalkProfileReview } from '../../api/walk-profile-client';
import { Button } from '../../ui/Button';
import { Input, Select } from '../../ui/Input';

interface Props {
  review: WalkProfileReview;
  setReview: (review: WalkProfileReview) => void;
  creating: boolean;
  createProfile: () => Promise<void>;
}

const deviceTypes = [
  'switch',
  'router',
  'firewall',
  'access-point',
  'host',
  'server',
  'voip-phone',
  'printer',
];

export const WalkProfileReviewForm: FC<Props> = ({
  review,
  setReview,
  creating,
  createProfile,
}) => {
  const { t } = useTranslation('pages');
  const update = (field: keyof WalkProfileReview['profile'], value: string) =>
    setReview({ ...review, profile: { ...review.profile, [field]: value } });
  const valid =
    /^[a-z][a-z0-9-]{1,47}$/.test(review.profile.role) &&
    [
      review.profile.deviceType,
      review.profile.vendor,
      review.profile.model,
      review.profile.platform,
    ].every((value) => value.trim().length > 0);

  return (
    <section
      className="stack-lg rounded-lg border border-status-info/30 bg-status-info/5 p-comfortable"
      data-testid="walk-profile-review"
    >
      <div>
        <h3 className="font-semibold text-text-primary">{t('walkAnalyzer.profile.reviewTitle')}</h3>
        <p className="text-sm text-text-muted">
          {t('walkAnalyzer.profile.reviewEvidence', {
            interfaces: review.profile.interfaceCount ?? 0,
            capabilities: review.profile.supportedSnmpData?.join(', ') ?? '',
          })}
        </p>
      </div>
      <div className="grid gap-default md:grid-cols-2">
        <Input
          label={t('walkAnalyzer.profile.role')}
          value={review.profile.role}
          onChange={(event) => update('role', event.target.value)}
        />
        <Select
          label={t('walkAnalyzer.profile.deviceType')}
          value={review.profile.deviceType}
          options={deviceTypes.map((value) => ({ value, label: value }))}
          onChange={(value) => update('deviceType', value)}
        />
        <Input
          label={t('walkAnalyzer.profile.vendor')}
          value={review.profile.vendor}
          onChange={(event) => update('vendor', event.target.value)}
        />
        <Input
          label={t('walkAnalyzer.profile.model')}
          value={review.profile.model}
          onChange={(event) => update('model', event.target.value)}
        />
        <Input
          label={t('walkAnalyzer.profile.platform')}
          value={review.profile.platform}
          onChange={(event) => update('platform', event.target.value)}
        />
        <Input
          label={t('walkAnalyzer.profile.software')}
          value={review.profile.software}
          onChange={(event) => update('software', event.target.value)}
        />
      </div>
      <Button
        type="button"
        tone="green"
        disabled={!valid || creating}
        loading={creating}
        onClick={() => void createProfile()}
        data-testid="walk-profile-create"
      >
        {t('walkAnalyzer.profile.createAction')}
      </Button>
    </section>
  );
};
