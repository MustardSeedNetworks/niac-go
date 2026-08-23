/**
 * Re-frame the React Flow canvas after React has committed a new layout.
 *
 * A fixed setTimeout races the dagre pass on a large graph: fitView measures
 * stale node geometry and the canvas lands at a wrong zoom with almost nothing
 * in view (D15). Two nested animation frames guarantee a committed paint first.
 *
 * Returns a cancel function so effects can clean up on unmount.
 */
export function reframeAfterPaint(fit: () => void): () => void {
  let inner = 0;
  const outer = window.requestAnimationFrame(() => {
    inner = window.requestAnimationFrame(fit);
  });

  return () => {
    window.cancelAnimationFrame(outer);
    window.cancelAnimationFrame(inner);
  };
}
