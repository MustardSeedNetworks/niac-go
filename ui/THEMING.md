# NIAC theme system

The NIAC UI uses semantic CSS variables, Tailwind utilities, and typed token
exports. The rendered CSS is authoritative; this document explains ownership
and constraints rather than duplicating every token value.

## Ownership

- `src/index.css` owns light and dark semantic variables and the Tailwind
  `@theme` mapping.
- `src/styles/theme.ts` exports typed token groups for TSX code.
- `src/styles/themeDeviceColors.ts` owns topology device, link, and protocol
  colors.
- `src/styles/themeModuleColors.ts` owns module accents.
- `src/hooks/useTheme.ts` owns theme selection and persistence.

Components should consume semantic utilities or exported role tokens. Do not
repeat raw brand or status hex values in component code.

## Brand and module roles

NIAC's primary brand color is indigo. Mustard is the shared cross-product
accent. Module colors distinguish topology, protocols, analysis, injection,
and configuration surfaces; they are accents for icons, badges, and legends,
not card backgrounds.

Status colors retain one meaning across themes:

- green for success;
- mustard for warning;
- red for danger; and
- blue for information.

Status must also have text or an icon. Color alone cannot carry meaning.

## Surfaces and text

Light and dark themes define page, raised, sunken, card, border, foreground,
muted, and inverse roles. Use the role that describes the element. Avoid
choosing a shade by visual coincidence.

Text and interactive controls must meet WCAG AA contrast. Focus indicators must
remain visible in both themes, and motion must respect
`prefers-reduced-motion`.

## Typography

Inter Variable is the UI font and JetBrains Mono Variable is the code font.
Both are bundled locally. Use the existing heading, body, label, caption, and
code classes instead of inventing component-specific scales.

## Theme switching

The theme hook applies the effective theme to the root element and exposes the
selected and resolved values:

```tsx
import { useTheme } from '../hooks/useTheme';

const { theme, setTheme, actualTheme } = useTheme();
```

Do not read or write theme storage directly from components.

## Review checklist

For a theme change:

1. update the owning semantic token;
2. verify light and dark rendering;
3. test keyboard focus and reduced motion;
4. check text and control contrast; and
5. run visual tests at supported desktop and mobile widths.

Chrome, Edge, and Safari are first-class browser targets. Firefox remains an
independent compatibility engine, and Brave receives a privacy-default smoke
test before release.
