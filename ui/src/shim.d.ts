/**
 * Type Shims
 *
 * Purpose: Provides TypeScript type declarations for modules that don't have
 * built-in type definitions. Prevents TS2882 errors when importing modules
 * without @types packages or with side-effect-only entry points.
 *
 * Usage: Automatically applied by TypeScript compiler, no explicit imports needed.
 */

// Side-effect-only font CSS imports — no exports needed.
declare module "@fontsource-variable/manrope";
declare module "@fontsource-variable/inter";
declare module "@fontsource-variable/jetbrains-mono";

/**
 * Dependency versions injected by vite.config.ts from package.json, so the
 * Settings > About panel cannot drift from what actually shipped (D17).
 */
declare const __REACT_VERSION__: string;
declare const __TYPESCRIPT_VERSION__: string;
