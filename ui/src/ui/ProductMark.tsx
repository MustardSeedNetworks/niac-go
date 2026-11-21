interface ProductMarkProps {
  /** `sm` is the mobile top bar, `md` the rail lockup, `lg` the sign-in card. */
  size?: 'sm' | 'md' | 'lg';
}

const SIZES = {
  sm: 'h-8 w-8 text-xs',
  md: 'h-9 w-9 text-sm',
  lg: 'h-12 w-12 text-base',
} as const;

/** The NIAC product mark: answers "which tool am I in". */
export function ProductMark({ size = 'md' }: ProductMarkProps) {
  return (
    <div
      data-testid="product-mark"
      className={`${SIZES[size]} rounded-[11px] bg-brand-primary flex-center`}
    >
      {/* allow-hardcoded: the monogram is a brand glyph, not translatable copy. */}
      <span className="figure font-extrabold tracking-tight text-on-brand">NI</span>
    </div>
  );
}
