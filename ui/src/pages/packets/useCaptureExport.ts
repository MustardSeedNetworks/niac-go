import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { exportSessionCapture } from '../../api/client';
import type { Packet } from '../../components/PacketList';
import { useErrorToast } from '../../hooks/useErrorToast';

/**
 * downloadBlob hands the viewer a file. Both inspector exports need it, and
 * neither can use a plain <a href> to the API: the bearer token lives in
 * memory and never reaches the browser's request for a navigation.
 */
function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * useFilteredJsonExport serialises what is on screen — the page's own packet
 * list after its display filter — as JSON. The pcapng export below is the
 * other half: what the daemon retained, whole.
 */
export function useFilteredJsonExport(filteredPackets: Packet[]) {
  return useCallback(() => {
    if (filteredPackets.length === 0) {
      return;
    }
    const stamp = new Date().toISOString().slice(0, 19).replace(/[:.]/g, '-');
    downloadBlob(
      new Blob([JSON.stringify(filteredPackets, null, 2)], { type: 'application/json' }),
      `packets-${stamp}.json`,
    );
  }, [filteredPackets]);
}

/**
 * useCaptureExport downloads the daemon's retained frames for a session as a
 * pcapng file.
 *
 * This is deliberately not the page's own packet list: the browser keeps the
 * last 100 frames truncated to 256 bytes for the hex pane, while the daemon's
 * ring keeps whole frames and the fabric decision behind each one — the part
 * Wireshark cannot infer from the wire.
 *
 * The page's display filter is not forwarded either. It is the UI's own
 * expression language, not BPF, and the endpoint's `filter` compiles through
 * libpcap. Narrowing the capture is Wireshark's job once it opens.
 */
export function useCaptureExport(sessionId: string | undefined) {
  const { t } = useTranslation('pages');
  const showError = useErrorToast();

  return useCallback(async () => {
    if (!sessionId) {
      return;
    }
    try {
      downloadBlob(await exportSessionCapture(sessionId), `niac-${sessionId}.pcapng`);
    } catch (err) {
      showError(err, t('packets.inspector.exportPcapError'));
    }
  }, [sessionId, showError, t]);
}
