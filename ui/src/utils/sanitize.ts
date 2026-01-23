/**
 * Sanitize general user input for programmatic string usage
 * (URL construction, localStorage keys, etc.).
 * Strips control characters and trims whitespace.
 */
export function sanitizeInput(input: string): string {
  let result = '';
  for (const char of input) {
    const code = char.charCodeAt(0);
    if (code >= 0x20 && code !== 0x7f) {
      result += char;
    }
  }
  return result.trim();
}

/**
 * Sanitize a hostname string for use in URL paths.
 * Allows alphanumeric, hyphens, dots, and underscores only.
 */
export function sanitizeHostname(hostname: string): string {
  return hostname.replace(/[^a-zA-Z0-9.\-_]/g, '');
}
