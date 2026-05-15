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
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <H2 className="mb-0 flex items-center gap-2">
          <FileInput className={`${iconSizes.lg} text-emerald-300`} />
          Import legacy config (Java DSL → YAML)
        </H2>
        <P className="text-sm text-gray-400">
          Paste a legacy <code>.cfg</code> file (the Java-DSL <code>device foo {'{ ... }'}</code>{' '}
          format) or upload one — same as <code>niac config export</code> on the CLI. The result is
          a normalised YAML you can save as a template, drop into a Git repo, or feed straight into
          a simulation.
        </P>

        <div className="flex flex-wrap items-center gap-3">
          <label className="cursor-pointer rounded bg-gray-800/60 px-3 py-1.5 text-xs font-medium text-gray-200 ring-1 ring-white/10 hover:bg-gray-800">
            Choose .cfg file…
            <input type="file" accept=".cfg,.conf,.txt" onChange={onPickFile} className="hidden" />
          </label>
          <SmallText className="text-gray-500">
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
          className="w-full rounded border border-white/5 bg-gray-950/60 p-3 font-mono text-xs text-gray-100 placeholder-gray-500 focus:border-emerald-400 focus:outline-none"
          aria-label="Legacy Java DSL config content"
        />

        <div className="flex flex-wrap items-center gap-3">
          <Button
            tone="green"
            disabled={busy || !content.trim()}
            onClick={() => void onImport()}
            title="POST the content to /api/v1/config/import?format=java-dsl. Returns normalised YAML; nothing is saved server-side."
          >
            {busy ? 'Converting…' : 'Convert to YAML'}
          </Button>
          {error && (
            <SmallText className="text-red-300" role="alert">
              {error}
            </SmallText>
          )}
        </div>

        {yaml && (
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <SmallText className="text-emerald-300">
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
            <pre className="max-h-64 overflow-auto rounded bg-gray-950/80 p-3 font-mono text-xs text-gray-200">
              {yaml}
            </pre>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
