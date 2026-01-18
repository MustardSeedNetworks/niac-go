import { memo } from "react";
import { SmallText } from "../ui/Typography";

interface StatBlockProps {
	label: string;
	value: string;
	helper: string;
}

/**
 * Stat block component for displaying key metrics
 */
export const StatBlock = memo(({ label, value, helper }: StatBlockProps) => (
	<div>
		<SmallText className="uppercase tracking-wide text-gray-400">
			{label}
		</SmallText>
		<p className="text-3xl font-bold text-white">{value}</p>
		<SmallText className="text-gray-300">{helper}</SmallText>
	</div>
));

StatBlock.displayName = "StatBlock";
