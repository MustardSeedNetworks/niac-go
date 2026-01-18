import { useCallback, useEffect, useRef, useState } from "react";

interface Options<T> {
	intervalMs?: number;
	transform?: (value: T) => T;
}

export function useApiResource<T>(
	fetcher: () => Promise<T>,
	deps: unknown[] = [],
	options: Options<T> = {},
) {
	const { intervalMs, transform } = options;
	const [data, setData] = useState<T | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<Error | null>(null);
	const timerRef = useRef<number | null>(null);
	const fetcherRef = useRef<() => Promise<T>>(fetcher);

	// Update fetcher ref when it changes
	useEffect(() => {
		fetcherRef.current = fetcher;
	}, [fetcher]);

	const run = useCallback(async () => {
		try {
			const result = await fetcherRef.current();
			setData(transform ? transform(result) : result);
			setError(null);
		} catch (err) {
			setError(err as Error);
		} finally {
			setLoading(false);
		}
	}, [transform]);

	useEffect(() => {
		let cancelled = false;

		const runWithCancellation = async () => {
			if (cancelled) {
				return;
			}
			await run();
		};

		void runWithCancellation();

		if (intervalMs) {
			timerRef.current = window.setInterval(() => {
				void run();
			}, intervalMs);
		}

		return () => {
			cancelled = true;
			if (timerRef.current) {
				clearInterval(timerRef.current);
				timerRef.current = null;
			}
		};
	}, [...deps, intervalMs, run]);

	return { data, loading, error, refetch: run } as const;
}
