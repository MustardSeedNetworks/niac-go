export function newDraftName(now = new Date()): string {
  const timestamp = now.toISOString().replaceAll(/[-:.TZ]/g, '');
  return `scenario-${timestamp}-${crypto.randomUUID()}`;
}
