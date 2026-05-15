import { Copy } from 'lucide-react';
import { type FC, useState } from 'react';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';

interface CloneDeviceModalProps {
  hostname: string;
  onClone: (newHostname: string) => void;
  onCancel: () => void;
}

export const CloneDeviceModal: FC<CloneDeviceModalProps> = ({ hostname, onClone, onCancel }) => {
  const [newHostname, setNewHostname] = useState(`${hostname}-copy`);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newHostname.trim()) {
      onClone(newHostname.trim());
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <button
        type="button"
        className="absolute inset-0 bg-black/70 backdrop-blur-sm"
        onClick={onCancel}
        aria-label="Close clone device modal"
      />
      <div
        className="mx-4 w-full max-w-md rounded-2xl border border-white/10 bg-gray-900/95 shadow-2xl"
        role="dialog"
        aria-modal="true"
      >
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          <div className="flex items-center gap-3 text-blue-400">
            <Copy className={iconSizes.xl} />
            <h2 className="text-lg font-semibold">Clone Device</h2>
          </div>
          <p className="text-gray-300">
            Create a copy of <strong>{hostname}</strong> with a new name.
          </p>
          <div>
            <label htmlFor="new-hostname" className="block text-sm font-medium text-gray-300 mb-2">
              New Hostname
            </label>
            <input
              id="new-hostname"
              type="text"
              value={newHostname}
              onChange={(e) => setNewHostname(e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
            />
          </div>
          <div className="flex justify-end gap-3 pt-2">
            <Button variant="outline" type="button" onClick={onCancel}>
              Cancel
            </Button>
            <Button tone="violet" type="submit" disabled={!newHostname.trim()}>
              Clone
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};
