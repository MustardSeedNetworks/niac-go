import { ChevronLeft, ChevronRight, ChevronsLeftRight } from "lucide-react";
import { type FC, memo, useCallback, useMemo } from "react";
import { Button } from "../../ui/Button";
import { Tag } from "../../ui/Tag";
import { SmallText } from "../../ui/Typography";

/**
 * Types for diff operations
 */
export type DiffType = "unchanged" | "added" | "removed" | "modified";

export interface DiffLine {
	lineNumber: number;
	content: string;
	type: DiffType;
	leftLineNumber?: number;
	rightLineNumber?: number;
}

export interface DiffBlock {
	id: string;
	leftLines: DiffLine[];
	rightLines: DiffLine[];
	type: DiffType;
	startLine: number;
}

export interface MergeDecision {
	blockId: string;
	choice: "left" | "right" | "both" | "none";
}

interface DiffViewerProps {
	leftContent: string;
	rightContent: string;
	leftLabel?: string;
	rightLabel?: string;
	mergeDecisions: Map<string, MergeDecision>;
	onMergeDecision: (blockId: string, choice: MergeDecision["choice"]) => void;
	showMergeControls?: boolean;
}

/**
 * Compute LCS-based diff between two content strings
 */
const isLcsMatch = (
	currentLcs: string | null,
	leftLines: string[],
	rightLines: string[],
	leftIdx: number,
	rightIdx: number,
): boolean =>
	currentLcs !== null &&
	leftIdx < leftLines.length &&
	rightIdx < rightLines.length &&
	leftLines[leftIdx] === currentLcs &&
	rightLines[rightIdx] === currentLcs;

const appendUnchangedLine = (
	blocks: DiffBlock[],
	line: DiffLine,
	blockId: number,
	leftIdx: number,
): number => {
	const lastBlock = blocks.at(-1);
	if (lastBlock && lastBlock.type === "unchanged") {
		lastBlock.leftLines.push(line);
		lastBlock.rightLines.push({ ...line });
		return blockId;
	}

	blocks.push({
		id: `block-${blockId}`,
		leftLines: [line],
		rightLines: [{ ...line }],
		type: "unchanged",
		startLine: leftIdx + 1,
	});

	return blockId + 1;
};

const collectChangedLines = (
	leftLines: string[],
	rightLines: string[],
	currentLcs: string | null,
	leftIdx: number,
	rightIdx: number,
): {
	leftChanged: DiffLine[];
	rightChanged: DiffLine[];
	nextLeftIdx: number;
	nextRightIdx: number;
} => {
	const leftChanged: DiffLine[] = [];
	const rightChanged: DiffLine[] = [];
	let nextLeftIdx = leftIdx;
	let nextRightIdx = rightIdx;

	while (
		nextLeftIdx < leftLines.length &&
		(currentLcs === null || leftLines[nextLeftIdx] !== currentLcs)
	) {
		leftChanged.push({
			lineNumber: nextLeftIdx + 1,
			content: leftLines[nextLeftIdx],
			type: "removed",
			leftLineNumber: nextLeftIdx + 1,
		});
		nextLeftIdx++;
	}

	while (
		nextRightIdx < rightLines.length &&
		(currentLcs === null || rightLines[nextRightIdx] !== currentLcs)
	) {
		rightChanged.push({
			lineNumber: nextRightIdx + 1,
			content: rightLines[nextRightIdx],
			type: "added",
			rightLineNumber: nextRightIdx + 1,
		});
		nextRightIdx++;
	}

	return { leftChanged, rightChanged, nextLeftIdx, nextRightIdx };
};

const getChangeType = (
	leftChanged: DiffLine[],
	rightChanged: DiffLine[],
): DiffType => {
	if (leftChanged.length > 0 && rightChanged.length > 0) {
		return "modified";
	}
	if (leftChanged.length > 0) {
		return "removed";
	}
	return "added";
};

function computeDiff(left: string, right: string): DiffBlock[] {
	const leftLines = left.split("\n");
	const rightLines = right.split("\n");

	const blocks: DiffBlock[] = [];
	let leftIdx = 0;
	let rightIdx = 0;
	let blockId = 0;

	// Simple line-by-line diff using longest common subsequence approach
	const lcs = computeLcs(leftLines, rightLines);
	let lcsIdx = 0;

	while (leftIdx < leftLines.length || rightIdx < rightLines.length) {
		const currentLcs = lcsIdx < lcs.length ? lcs[lcsIdx] : null;

		if (isLcsMatch(currentLcs, leftLines, rightLines, leftIdx, rightIdx)) {
			// Unchanged line
			const line: DiffLine = {
				lineNumber: leftIdx + 1,
				content: leftLines[leftIdx],
				type: "unchanged",
				leftLineNumber: leftIdx + 1,
				rightLineNumber: rightIdx + 1,
			};

			blockId = appendUnchangedLine(blocks, line, blockId, leftIdx);

			leftIdx++;
			rightIdx++;
			lcsIdx++;
		} else {
			const { leftChanged, rightChanged, nextLeftIdx, nextRightIdx } =
				collectChangedLines(
					leftLines,
					rightLines,
					currentLcs,
					leftIdx,
					rightIdx,
				);
			leftIdx = nextLeftIdx;
			rightIdx = nextRightIdx;

			if (leftChanged.length > 0 || rightChanged.length > 0) {
				const type = getChangeType(leftChanged, rightChanged);
				blocks.push({
					id: `block-${blockId++}`,
					leftLines: leftChanged,
					rightLines: rightChanged,
					type,
					startLine: Math.min(
						leftChanged[0]?.lineNumber ?? Number.POSITIVE_INFINITY,
						rightChanged[0]?.lineNumber ?? Number.POSITIVE_INFINITY,
					),
				});
			}
		}
	}

	return blocks;
}

/**
 * Compute longest common subsequence of lines
 */
function computeLcs(left: string[], right: string[]): string[] {
	const m = left.length;
	const n = right.length;

	// Build LCS length table
	const dp: number[][] = Array.from({ length: m + 1 }, () =>
		Array.from({ length: n + 1 }, () => 0),
	);

	for (let i = 1; i <= m; i++) {
		for (let j = 1; j <= n; j++) {
			if (left[i - 1] === right[j - 1]) {
				dp[i][j] = dp[i - 1][j - 1] + 1;
			} else {
				dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1]);
			}
		}
	}

	// Backtrack to find LCS
	const lcs: string[] = [];
	let i = m;
	let j = n;

	while (i > 0 && j > 0) {
		if (left[i - 1] === right[j - 1]) {
			lcs.unshift(left[i - 1]);
			i--;
			j--;
		} else if (dp[i - 1][j] > dp[i][j - 1]) {
			i--;
		} else {
			j--;
		}
	}

	return lcs;
}

/**
 * Get styling classes for diff type
 */
function getDiffStyles(type: DiffType): {
	bg: string;
	border: string;
	text: string;
} {
	switch (type) {
		case "added":
			return {
				bg: "bg-green-500/10",
				border: "border-green-500/30",
				text: "text-green-300",
			};
		case "removed":
			return {
				bg: "bg-red-500/10",
				border: "border-red-500/30",
				text: "text-red-300",
			};
		case "modified":
			return {
				bg: "bg-yellow-500/10",
				border: "border-yellow-500/30",
				text: "text-yellow-300",
			};
		default:
			return {
				bg: "",
				border: "border-transparent",
				text: "text-gray-300",
			};
	}
}

/**
 * Individual diff line component
 */
const DiffLineComponent: FC<{
	line: DiffLine;
	side: "left" | "right";
}> = memo(({ line, side }) => {
	const styles = getDiffStyles(line.type);
	const lineNum = side === "left" ? line.leftLineNumber : line.rightLineNumber;

	return (
		<div className={`flex ${styles.bg} border-l-2 ${styles.border}`}>
			<span className="w-12 flex-shrink-0 px-2 py-0.5 text-right text-xs text-gray-500 select-none border-r border-white/5">
				{lineNum ?? ""}
			</span>
			<span className="px-1 py-0.5 text-xs text-gray-400 select-none w-4">
				{line.type === "added" ? "+" : line.type === "removed" ? "-" : " "}
			</span>
			<pre
				className={`flex-1 px-2 py-0.5 text-sm font-mono ${styles.text} overflow-x-auto whitespace-pre`}
			>
				{line.content || " "}
			</pre>
		</div>
	);
});

DiffLineComponent.displayName = "DiffLineComponent";

/**
 * Diff block with merge controls
 */
const DiffBlockComponent: FC<{
	block: DiffBlock;
	decision?: MergeDecision;
	onDecision: (choice: MergeDecision["choice"]) => void;
	showMergeControls: boolean;
}> = memo(({ block, decision, onDecision, showMergeControls }) => {
	const isChanged = block.type !== "unchanged";
	const maxLines = Math.max(block.leftLines.length, block.rightLines.length);

	// Pad lines to equal length for alignment
	const paddedLeftLines = [...block.leftLines];
	const paddedRightLines = [...block.rightLines];

	while (paddedLeftLines.length < maxLines) {
		paddedLeftLines.push({
			lineNumber: -1,
			content: "",
			type: "unchanged",
		});
	}

	while (paddedRightLines.length < maxLines) {
		paddedRightLines.push({
			lineNumber: -1,
			content: "",
			type: "unchanged",
		});
	}

	return (
		<div
			className={`${isChanged ? "border border-white/10 rounded-lg overflow-hidden mb-2" : ""}`}
		>
			{/* Merge controls for changed blocks */}
			{isChanged && showMergeControls && (
				<div className="flex items-center justify-center gap-2 py-2 px-4 bg-gray-900/80 border-b border-white/10">
					<SmallText className="text-gray-400 mr-2">Accept:</SmallText>
					<Button
						size="sm"
						variant={decision?.choice === "left" ? undefined : "outline"}
						tone={decision?.choice === "left" ? "violet" : undefined}
						onClick={() => onDecision("left")}
						leftIcon={<ChevronLeft className="h-3 w-3" />}
					>
						Left
					</Button>
					<Button
						size="sm"
						variant={decision?.choice === "both" ? undefined : "outline"}
						tone={decision?.choice === "both" ? "violet" : undefined}
						onClick={() => onDecision("both")}
						leftIcon={<ChevronsLeftRight className="h-3 w-3" />}
					>
						Both
					</Button>
					<Button
						size="sm"
						variant={decision?.choice === "right" ? undefined : "outline"}
						tone={decision?.choice === "right" ? "violet" : undefined}
						onClick={() => onDecision("right")}
						leftIcon={<ChevronRight className="h-3 w-3" />}
					>
						Right
					</Button>
					{decision && (
						<Tag colorScheme="purple" className="ml-2 text-xs">
							{decision.choice.toUpperCase()}
						</Tag>
					)}
				</div>
			)}

			{/* Side-by-side diff display */}
			<div className="grid grid-cols-2 divide-x divide-white/10">
				{/* Left panel */}
				<div className="overflow-x-auto">
					{paddedLeftLines.map((line, idx) => (
						<DiffLineComponent
							key={`left-${block.id}-${idx}`}
							line={line}
							side="left"
						/>
					))}
				</div>

				{/* Right panel */}
				<div className="overflow-x-auto">
					{paddedRightLines.map((line, idx) => (
						<DiffLineComponent
							key={`right-${block.id}-${idx}`}
							line={line}
							side="right"
						/>
					))}
				</div>
			</div>
		</div>
	);
});

DiffBlockComponent.displayName = "DiffBlockComponent";

/**
 * Side-by-side diff viewer component
 */
export const DiffViewer: FC<DiffViewerProps> = ({
	leftContent,
	rightContent,
	leftLabel = "Original",
	rightLabel = "Modified",
	mergeDecisions,
	onMergeDecision,
	showMergeControls = true,
}) => {
	// Compute diff blocks
	const diffBlocks = useMemo(
		() => computeDiff(leftContent, rightContent),
		[leftContent, rightContent],
	);

	// Count changes
	const stats = useMemo(() => {
		let additions = 0;
		let deletions = 0;
		let modifications = 0;

		for (const block of diffBlocks) {
			if (block.type === "added") {
				additions += block.rightLines.length;
			} else if (block.type === "removed") {
				deletions += block.leftLines.length;
			} else if (block.type === "modified") {
				modifications++;
			}
		}

		return { additions, deletions, modifications };
	}, [diffBlocks]);

	const handleDecision = useCallback(
		(blockId: string) => (choice: MergeDecision["choice"]) => {
			onMergeDecision(blockId, choice);
		},
		[onMergeDecision],
	);

	if (!(leftContent || rightContent)) {
		return (
			<div className="flex items-center justify-center h-64 text-gray-400">
				<SmallText>Upload files to compare</SmallText>
			</div>
		);
	}

	return (
		<div className="space-y-4">
			{/* Stats bar */}
			<div className="flex items-center gap-4 flex-wrap">
				<SmallText className="text-gray-400">Changes:</SmallText>
				<Tag colorScheme="green" className="text-xs">
					+{stats.additions} additions
				</Tag>
				<Tag colorScheme="red" className="text-xs">
					-{stats.deletions} deletions
				</Tag>
				<Tag colorScheme="yellow" className="text-xs">
					{stats.modifications} modified blocks
				</Tag>
				{showMergeControls && (
					<SmallText className="text-gray-400 ml-auto">
						{mergeDecisions.size} /{" "}
						{diffBlocks.filter((b) => b.type !== "unchanged").length} decisions
						made
					</SmallText>
				)}
			</div>

			{/* Column headers */}
			<div className="grid grid-cols-2 gap-px bg-gray-800 rounded-t-lg overflow-hidden">
				<div className="bg-gray-900/90 px-4 py-2">
					<SmallText className="text-gray-300 font-semibold">
						{leftLabel}
					</SmallText>
				</div>
				<div className="bg-gray-900/90 px-4 py-2">
					<SmallText className="text-gray-300 font-semibold">
						{rightLabel}
					</SmallText>
				</div>
			</div>

			{/* Diff content */}
			<div className="rounded-lg border border-white/10 bg-gray-950/50 overflow-hidden max-h-[500px] overflow-y-auto">
				{diffBlocks.map((block) => (
					<DiffBlockComponent
						key={block.id}
						block={block}
						decision={mergeDecisions.get(block.id)}
						onDecision={handleDecision(block.id)}
						showMergeControls={showMergeControls}
					/>
				))}
			</div>
		</div>
	);
};

/**
 * Generate merged content based on decisions
 */
const getDecisionLines = (
	block: DiffBlock,
	decision?: MergeDecision,
): { leftLines: DiffLine[]; rightLines: DiffLine[] } => {
	if (!decision || decision.choice === "none") {
		if (block.type === "added") {
			return { leftLines: [], rightLines: block.rightLines };
		}
		return { leftLines: block.leftLines, rightLines: [] };
	}

	if (decision.choice === "left") {
		return { leftLines: block.leftLines, rightLines: [] };
	}

	if (decision.choice === "right") {
		return { leftLines: [], rightLines: block.rightLines };
	}

	return { leftLines: block.leftLines, rightLines: block.rightLines };
};

export function generateMergedContent(
	leftContent: string,
	rightContent: string,
	decisions: Map<string, MergeDecision>,
): string {
	const diffBlocks = computeDiff(leftContent, rightContent);
	const mergedLines: string[] = [];

	for (const block of diffBlocks) {
		if (block.type === "unchanged") {
			// Include unchanged lines
			for (const line of block.leftLines) {
				mergedLines.push(line.content);
			}
		} else {
			const decisionLines = getDecisionLines(block, decisions.get(block.id));
			for (const line of decisionLines.leftLines) {
				mergedLines.push(line.content);
			}
			for (const line of decisionLines.rightLines) {
				mergedLines.push(line.content);
			}
		}
	}

	return mergedLines.join("\n");
}

export { computeDiff };
