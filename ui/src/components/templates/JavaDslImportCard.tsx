import { FileInput } from 'lucide-react';
import { type FC, useState } from 'react';
import { importConfig } from '../../api/client';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { H2, P, SmallText } from '../../ui/Typography';

/**
 * JavaDslImportCard converts a legacy Java-DSL `.cfg` payload to YAML via
 * POST /api/v1/config/import?format=java-dsl. Mirrors `niac config export`.
 *
 * Lives on TemplatesPage so users importing legacy configs land somewhere
 * obvious when they're starting from a non-YAML source.
 */
export const JavaDslImportCard: FC = () => {
  const [content, setContent] = useState('');
  const [yaml, setYaml] = useState<string | null>(null);
  const [deviceCount, setDeviceCount] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const onImport = async () => {
    setBusy(true);
    setError(null);
    setYaml(null);
    try {
      const result = await importConfig({ format: 'java-dsl', content });
      setYaml(result.yaml);
      setDeviceCount(result.devices);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const onPickFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const text = await file.text();
    setContent(text);
    setYaml(null);
    setError(null);
  };

  const downloadYaml = () => {
    if (!yaml) return;
    const blob = new Blob([yaml], { type: 'application/x-yaml' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'imported-config.yaml';
    link.click();
    URL.revokeObjectURL(url);
  };

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack-lg">
        <H2 className="flex items-center gap-compact">
          <FileInput className={`${iconSizes.lg} text-status-success`} />
          Import legacy config (Java DSL → YAML)
        </H2>
        <P className="text-sm text-text-muted">
          Paste a legacy <code>.cfg</code> file (the Java-DSL <code>device foo {'{ ... }'}</code>{' '}
          format) or upload one — same as <code>niac config export</code> on the CLI. The result is
          a normalised YAML you can save as a template, drop into a Git repo, or feed straight into
          a simulation.
        </P>

        <div className="flex flex-wrap items-center gap-default">
          <label className="cursor-pointer rounded bg-bg-elevated/60 px-3 py-compact-md text-xs font-medium text-text-primary ring-1 ring-knob/10 hover:bg-bg-elevated">
            Choose .cfg file…
            <input type="file" accept=".cfg,.conf,.txt" onChange={onPickFile} className="hidden" />
          </label>
          <SmallText className="text-text-muted">
            Or paste below ({content.length.toLocaleString()} chars)
          </SmallText>
        </div>

        <textarea
          value={content}
          onChange={(e) => {
            setContent(e.target.value);
            setYaml(null);
          }}
          placeholder="device my-router {&#10;  type = router&#10;  ip = 192.168.1.1&#10;  ...&#10;}"
          rows={10}
          className="w-full rounded border border-surface-border bg-bg-base/60 pad-sm font-mono text-xs text-text-primary placeholder:text-text-muted focus:border-status-success focus:outline-none"
          aria-label="Legacy Java DSL config content"
        />

        <div className="flex flex-wrap items-center gap-default">
          <Button
            tone="green"
            disabled={busy || !content.trim()}
            onClick={() => void onImport()}
            title="POST the content to /api/v1/config/import?format=java-dsl. Returns normalised YAML; nothing is saved server-side."
          >
            {busy ? 'Converting…' : 'Convert to YAML'}
          </Button>
          {error && (
            <SmallText className="text-status-error" role="alert">
              {error}
            </SmallText>
          )}
        </div>

        {yaml && (
          <div className="stack-sm">
            <div className="flex-between">
              <SmallText className="text-status-success">
                ✓ Converted ({deviceCount} {deviceCount === 1 ? 'device' : 'devices'})
              </SmallText>
              <Button
                variant="outline"
                onClick={downloadYaml}
                title="Save the converted YAML to your machine. Use it as a starter template or commit it to your config repo."
              >
                Download YAML
              </Button>
            </div>
            <pre className="max-h-64 overflow-auto rounded bg-bg-base/80 pad-sm font-mono text-xs text-text-primary">
              {yaml}
            </pre>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
