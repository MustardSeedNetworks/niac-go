import { type FC, useMemo, useState } from 'react';
import { fetchSynthesizeWalkModels, synthesizeWalk } from '../../api/client';
import { isApiError } from '../../api/errors';
import type { ModelDescriptor } from '../../api/types';
import { useApiResource } from '../../hooks/useApiResource';
import { Button } from '../../ui/Button';
import { getErrorMessage } from '../../utils/format';
import { FormField } from '../form';
import { selectClassName } from './types';

export interface SynthesizeWalkControlProps {
  /** Canonical device name — the POST target and the synthesized walk's filename. */
  hostname: string;
  /** True for an unsaved device; synthesize 404s until the device exists server-side. */
  disabled: boolean;
  /** Called with the library-relative walkPath the daemon attached, so the caller can mirror it into snmpAgent.walkFile. */
  onSynthesized: (walkPath: string) => void;
}

interface ModelOption {
  index: number;
  model: ModelDescriptor;
}

interface VendorGroup {
  vendor: string;
  options: ModelOption[];
}

/**
 * Groups the flat catalog by vendor for <optgroup> rendering. The
 * server already returns the catalog sorted by (vendor, type, label)
 * (synth.AllModels), so preserving encounter order here keeps both the
 * vendor groups and the options within each group stably ordered
 * without a client-side sort.
 */
function groupByVendor(models: ModelDescriptor[]): VendorGroup[] {
  const groups = new Map<string, VendorGroup>();
  models.forEach((model, index) => {
    const group = groups.get(model.vendor);
    if (group) {
      group.options.push({ index, model });
    } else {
      groups.set(model.vendor, { vendor: model.vendor, options: [{ index, model }] });
    }
  });
  return [...groups.values()];
}

/**
 * SynthesizeWalkControl — generates a baseline SNMP walk for this device
 * from a vendor/model profile, giving it a working SNMP responder
 * without capturing a walk from real hardware. Backed by
 * GET /api/v1/synthesize-walk/models (the catalog) and POST
 * /api/v1/devices/{hostname}/synthesize-walk (the generate action); see
 * docs/design/2026-05-baseline-walk-synthesis.md for the full contract.
 *
 * The option value is the model's index into the fetched array rather
 * than an encoded "vendor model" string, so lookup on generate never has
 * to split/parse a label that may itself contain spaces.
 */
export const SynthesizeWalkControl: FC<SynthesizeWalkControlProps> = ({
  hostname,
  disabled,
  onSynthesized,
}) => {
  const { data: models, error: modelsError } = useApiResource(fetchSynthesizeWalkModels, []);
  const [selected, setSelected] = useState('');
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const groups = useMemo(() => groupByVendor(models ?? []), [models]);

  const handleGenerate = async () => {
    const model = selected === '' ? undefined : models?.[Number(selected)];
    if (!model) {
      return;
    }

    setGenerating(true);
    setError(null);
    setSuccess(null);
    try {
      const res = await synthesizeWalk(hostname, { vendor: model.vendor, model: model.model });
      onSynthesized(res.walkPath);
      setSuccess(
        `Generated ${model.label} — ${res.oidCount} OIDs, ${res.sizeBytes} bytes${
          res.originalPreserved ? '; original preserved' : ''
        }`,
      );
    } catch (err) {
      if (isApiError(err) && err.code === 'unsupported_combo') {
        setError('This vendor has no profile for this device type.');
      } else {
        setError(getErrorMessage(err));
      }
    } finally {
      setGenerating(false);
    }
  };

  return (
    <FormField
      label="Synthesize Baseline Walk"
      helpText="Generate a baseline SNMP walk from a vendor/model profile instead of capturing one from real hardware"
    >
      <div className="stack-sm" data-testid="synthesize-walk-control">
        <div className="flex flex-wrap items-center gap-default">
          <select
            value={selected}
            onChange={(e) => {
              setSelected(e.target.value);
              setError(null);
              setSuccess(null);
            }}
            disabled={disabled || generating || !models}
            aria-label="Baseline walk model"
            data-testid="synthesize-walk-model-select"
            className={selectClassName}
          >
            <option value="">Select a model...</option>
            {groups.map((group) => (
              <optgroup key={group.vendor} label={group.vendor}>
                {group.options.map(({ index, model }) => (
                  <option key={`${group.vendor}-${model.model}`} value={index}>
                    {model.label}
                  </option>
                ))}
              </optgroup>
            ))}
          </select>
          <Button
            disabled={disabled || generating || selected === ''}
            onClick={() => void handleGenerate()}
            data-testid="synthesize-walk-generate"
          >
            {generating ? 'Generating…' : 'Generate walk'}
          </Button>
        </div>
        {disabled && (
          <p className="text-xs text-text-muted">Save the device before generating a walk.</p>
        )}
        {!disabled && modelsError && (
          <p className="text-xs text-status-error" role="alert">
            Could not load device profiles: {getErrorMessage(modelsError)}
          </p>
        )}
        {error && (
          <p className="text-xs text-status-error" role="alert">
            {error}
          </p>
        )}
        {success && <p className="text-xs text-status-success">{success}</p>}
      </div>
    </FormField>
  );
};
