import { FileCog, LineChart, PlugZap } from "lucide-react";
import { type FC, useState } from "react";
import {
	fetchFiles,
	fetchHistory,
	fetchReplayStatus,
	startReplay,
	stopReplay,
} from "../api/client";
import type { ReplayRequest } from "../api/types";
import { POLL_INTERVALS } from "../constants/polling";
import { useApiResource } from "../hooks/useApiResource";
import { Button } from "../ui/Button";
import { Card, CardContent } from "../ui/Card";
import { H2, P, SmallText } from "../ui/Typography";
import { copyToClipboard, fileToBase64 } from "../utils/file";
import {
	formatBytes,
	formatNumber,
	formatTime,
	getErrorMessage,
} from "../utils/format";

/**
 * Format duration string with fallback
 */
function formatDuration(value: string): string {
	return value || "—";
}

/**
 * Format relative time from timestamp
 */
function formatRelativeTime(timestamp: string): string {
	if (!timestamp) {
		return "—";
	}
	const diff = Date.now() - new Date(timestamp).getTime();
	if (diff < 0) {
		return "just now";
	}
	const seconds = Math.floor(diff / 1000);
	if (seconds < 60) {
		return `${seconds}s ago`;
	}
	const minutes = Math.floor(seconds / 60);
	if (minutes < 60) {
		return `${minutes}m ago`;
	}
	const hours = Math.floor(minutes / 60);
	if (hours < 24) {
		return `${hours}h ago`;
	}
	const days = Math.floor(hours / 24);
	return `${days}d ago`;
}

/**
 * Analysis Page - Capture & Walk Analysis
 *
 * Replay PCAPs, inspect SNMP walks, and publish bundles directly from the UI.
 */
export const AnalysisPage: FC = () => {
	const { data: history } = useApiResource(fetchHistory, [], {
		intervalMs: POLL_INTERVALS.SLOW,
	});

	return (
		<div className="space-y-6">
			<Card className="border-white/5 bg-gray-900/70">
				<CardContent className="space-y-4">
					<H2 className="mb-0 flex items-center gap-2">
						<LineChart className="h-5 w-5 text-emerald-300" />
						Capture & walk analysis
					</H2>
					<P>
						Replay PCAP files, filter packets, and bundle the results with SNMP
						walk exports. Every run can be published as downloadable evidence
						for troubleshooting or demo handoffs.
					</P>
					<div className="space-y-3 text-sm text-gray-300">
						{(history ?? []).slice(0, 5).map((item) => (
							<div
								key={item.id}
								className="rounded-lg border border-white/5 bg-gray-950/50 p-3"
							>
								<p className="text-white font-semibold">{item.configName}</p>
								<SmallText className="text-gray-400">
									{formatTime(item.startedAt)} · duration{" "}
									{formatDuration(item.duration)} · RX{" "}
									{formatNumber(item.packetsReceived)} · TX{" "}
									{formatNumber(item.packetsSent)}
								</SmallText>
							</div>
						))}
						{history?.length === 0 && (
							<SmallText className="text-gray-400">
								No captured runs yet.
							</SmallText>
						)}
					</div>
					<div className="flex flex-wrap gap-3">
						<Button tone="violet" leftIcon={<LineChart className="h-4 w-4" />}>
							Open analyzer
						</Button>
						<Button
							variant="outline"
							leftIcon={<FileCog className="h-4 w-4" />}
						>
							Export bundle
						</Button>
					</div>
				</CardContent>
			</Card>
			<ReplayPanel />
		</div>
	);
};

/**
 * Replay Panel - PCAP replay controls
 */
const ReplayPanel: FC = () => {
	const {
		data: status,
		loading,
		error,
	} = useApiResource(fetchReplayStatus, [], { intervalMs: 8000 });
	const { data: pcaps } = useApiResource(() => fetchFiles("pcaps"), [], {
		intervalMs: 45000,
	});
	const [pcapPath, setPcapPath] = useState("");
	const [loopMs, setLoopMs] = useState("");
	const [scale, setScale] = useState("");
	const [uploadFile, setUploadFile] = useState<File | null>(null);
	const [fileInputKey, setFileInputKey] = useState(0);
	const [busy, setBusy] = useState(false);
	const [message, setMessage] = useState<{
		tone: "success" | "error";
		text: string;
	} | null>(null);

	const handleStart = async () => {
		if (busy) {
			return;
		}
		const effectiveName = pcapPath.trim() || uploadFile?.name || "";
		if (!effectiveName) {
			setMessage({
				tone: "error",
				text: "Provide a PCAP path or upload a capture",
			});
			return;
		}
		setBusy(true);
		setMessage(null);
		try {
			const payload: ReplayRequest = { file: effectiveName };
			if (loopMs.trim()) {
				payload.loopMs = Number(loopMs);
			}
			if (scale.trim()) {
				payload.scale = Number(scale);
			}
			if (uploadFile) {
				payload.data = await fileToBase64(uploadFile);
			}
			await startReplay(payload);
			setMessage({ tone: "success", text: "Replay started" });
			if (uploadFile) {
				setUploadFile(null);
				setFileInputKey((value) => value + 1);
			}
		} catch (err) {
			setMessage({ tone: "error", text: getErrorMessage(err) });
		} finally {
			setBusy(false);
		}
	};

	const handleStop = async () => {
		if (busy) {
			return;
		}

		if (!window.confirm("Are you sure you want to stop the replay?")) {
			return;
		}

		setBusy(true);
		setMessage(null);
		try {
			await stopReplay();
			setMessage({ tone: "success", text: "Replay stopped" });
		} catch (err) {
			setMessage({ tone: "error", text: getErrorMessage(err) });
		} finally {
			setBusy(false);
		}
	};

	return (
		<Card className="border-white/5 bg-gray-900/70">
			<CardContent className="space-y-4">
				<H2 className="mb-0 flex items-center gap-2">
					<PlugZap className="h-5 w-5 text-pink-300" />
					Packet replay
				</H2>
				<P className="text-gray-300">
					Point NIAC at a PCAP file to replay capture traffic through the live
					interface. Replay honors loop timing and scaling so you can rapidly
					reproduce demos without leaving the Web UI.
				</P>
				{loading && (
					<SmallText className="text-gray-400">
						Checking replay engine…
					</SmallText>
				)}
				{error && (
					<SmallText className="text-red-400">
						Unable to read replay status: {error.message}
					</SmallText>
				)}
				<div className="grid gap-4 md:grid-cols-3">
					<div>
						<label
							htmlFor="pcap-file-path"
							className="block text-sm text-gray-400"
						>
							PCAP file
						</label>
						<input
							id="pcap-file-path"
							type="text"
							className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 font-mono text-sm text-white focus:border-violet-400 focus:outline-none"
							placeholder="/path/to/capture.pcap"
							value={pcapPath}
							onChange={(event) => setPcapPath(event.target.value)}
							aria-describedby="pcap-path-help"
						/>
					</div>
					<div>
						<label
							htmlFor="loop-interval"
							className="block text-sm text-gray-400"
						>
							Loop interval (ms)
						</label>
						<input
							id="loop-interval"
							type="number"
							className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
							placeholder="0"
							value={loopMs}
							onChange={(event) => setLoopMs(event.target.value)}
							aria-describedby="loop-help"
						/>
					</div>
					<div>
						<label htmlFor="time-scale" className="block text-sm text-gray-400">
							Time scale
						</label>
						<input
							id="time-scale"
							type="number"
							step="0.1"
							className="mt-1 w-full rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white focus:border-violet-400 focus:outline-none"
							placeholder="1.0"
							value={scale}
							onChange={(event) => setScale(event.target.value)}
							aria-describedby="scale-help"
						/>
					</div>
				</div>
				{message && (
					<SmallText
						className={
							message.tone === "success" ? "text-emerald-300" : "text-red-400"
						}
						role="alert"
						aria-live="polite"
					>
						{message.text}
					</SmallText>
				)}
				<div className="grid gap-4 md:grid-cols-2">
					<div>
						<label
							htmlFor="pcap-file-upload"
							className="block text-sm text-gray-400"
						>
							Upload PCAP
						</label>
						<input
							id="pcap-file-upload"
							key={fileInputKey}
							type="file"
							accept=".pcap,.pcapng,application/vnd.tcpdump.pcap"
							className="mt-1 w-full cursor-pointer rounded-lg border border-dashed border-white/10 bg-gray-950/40 p-2 text-sm text-white file:mr-3 file:rounded-md file:border-0 file:bg-violet-600 file:px-3 file:py-1 file:text-sm file:font-medium"
							onChange={(event) => {
								const file = event.target.files?.[0];
								if (!file) {
									setUploadFile(null);
									return;
								}

								const MaxSize = 100 * 1024 * 1024;
								if (file.size > MaxSize) {
									setMessage({
										tone: "error",
										text: `PCAP file too large. Maximum size is ${formatBytes(MaxSize)}`,
									});
									event.target.value = "";
									return;
								}

								if (!file.name.match(/\.(pcap|pcapng)$/i)) {
									setMessage({
										tone: "error",
										text: "Please select a PCAP file (.pcap or .pcapng)",
									});
									event.target.value = "";
									return;
								}

								setUploadFile(file);
							}}
							disabled={busy}
						/>
						<SmallText className="text-gray-500">
							If the server cannot access your filesystem, upload a capture
							directly from the browser.
						</SmallText>
					</div>
					{uploadFile && (
						<div className="rounded-lg border border-white/10 bg-gray-950/40 p-3 text-sm text-gray-300">
							<p className="font-semibold text-white">{uploadFile.name}</p>
							<SmallText className="text-gray-400">
								{formatBytes(uploadFile.size)}
							</SmallText>
						</div>
					)}
				</div>
				<div className="flex flex-wrap gap-3">
					<Button
						tone="violet"
						disabled={!(pcapPath.trim() || uploadFile) || busy}
						onClick={handleStart}
					>
						{busy ? "Working…" : "Start replay"}
					</Button>
					<Button
						variant="outline"
						disabled={busy || !status?.running}
						onClick={handleStop}
					>
						Stop replay
					</Button>
				</div>
				{pcaps && pcaps.length > 0 && (
					<div className="space-y-2">
						<SmallText className="text-gray-400">Discovered captures</SmallText>
						<div className="max-h-48 space-y-1 overflow-y-auto rounded-xl border border-white/10 bg-gray-950/50 p-2 text-sm text-gray-300">
							{pcaps.map((file) => (
								<div
									key={file.path}
									className="flex items-center justify-between gap-2 rounded-lg border border-white/5 bg-gray-900/50 px-3 py-2"
								>
									<div>
										<p className="text-white">{file.name}</p>
										<SmallText className="text-gray-500">{file.path}</SmallText>
									</div>
									<div className="flex gap-2">
										<Button
											size="sm"
											variant="outline"
											onClick={() => setPcapPath(file.path)}
										>
											Use path
										</Button>
										<Button
											size="sm"
											variant="ghost"
											onClick={() => copyToClipboard(file.path)}
										>
											Copy
										</Button>
									</div>
								</div>
							))}
						</div>
					</div>
				)}
				{status && (
					<div className="rounded-lg border border-white/10 bg-gray-950/50 p-3 text-sm text-gray-300">
						<p className="font-semibold text-white">
							{status.running ? "Running" : "Idle"}
						</p>
						{status.file && (
							<p className="font-mono text-xs text-gray-400">{status.file}</p>
						)}
						{status.running && status.startedAt && (
							<SmallText className="text-gray-400">
								Started {formatRelativeTime(status.startedAt)}
							</SmallText>
						)}
					</div>
				)}
			</CardContent>
		</Card>
	);
};

export default AnalysisPage;
