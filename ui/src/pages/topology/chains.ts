/**
 * Moves each chain of pass-through devices onto one rank, in chain order.
 *
 * A pass-through device has exactly two neighbours once leaves are packed: a
 * switch in a daisy chain or an industrial ring. Dagre ranks every hop one
 * level deeper, so the manufacturing pack's 18-switch ring hung down as two
 * columns about nine ranks deep and the graph came out nine times taller than
 * it was wide (#2106). A chain is one thing, the way a rank is, and laid out
 * as one it wraps like any other wide rank. It takes the rank of its
 * shallowest device, just below whatever it hangs off.
 */
export function foldChains(ranks: string[][], skeleton: Map<string, string[]>): string[][] {
  const passThrough = (name: string): boolean => skeleton.get(name)?.length === 2;
  const rankIndex = new Map<string, number>();
  ranks.forEach((rank, index) => {
    for (const name of rank) rankIndex.set(name, index);
  });
  const depth = (name: string): number => rankIndex.get(name) ?? Number.POSITIVE_INFINITY;

  const chains: string[][] = [];
  const seen = new Set<string>();
  for (const name of ranks.flat()) {
    if (seen.has(name) || !passThrough(name)) continue;
    // Gather the run, then walk it from an end so the order follows the links.
    const run = new Set<string>([name]);
    const queue = [name];
    for (let next = queue.pop(); next !== undefined; next = queue.pop()) {
      for (const peer of skeleton.get(next) ?? []) {
        if (passThrough(peer) && !run.has(peer)) {
          run.add(peer);
          queue.push(peer);
        }
      }
    }
    const ends = [...run].filter((member) =>
      (skeleton.get(member) ?? []).some((peer) => !run.has(peer)),
    );
    // A ring with nothing else attached has no end; any member starts it.
    let cursor: string | undefined = [...(ends.length > 0 ? ends : run)].sort(
      (left, right) => depth(left) - depth(right),
    )[0];
    const chain: string[] = [];
    while (cursor !== undefined) {
      chain.push(cursor);
      seen.add(cursor);
      cursor = (skeleton.get(cursor) ?? []).find((peer) => run.has(peer) && !seen.has(peer));
    }
    if (chain.length > 1) chains.push(chain);
  }
  if (chains.length === 0) return ranks;

  const folded = new Set(chains.flat());
  const result = ranks.map((rank) => rank.filter((name) => !folded.has(name)));
  for (const chain of chains) {
    const target = Math.min(...chain.map(depth));
    result[target]?.push(...chain);
  }
  return result.filter((rank) => rank.length > 0);
}
