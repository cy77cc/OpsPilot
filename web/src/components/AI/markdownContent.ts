export function normalizeMarkdownContent(content: string): string {
  const value = content || '';
  if (!value.includes('\\')) {
    return value;
  }

  // Historical runs may persist escaped line breaks as literal "\n".
  // Normalize them before markdown render/copy, but avoid touching content
  // that already contains real line breaks.
  if (value.includes('\n') || value.includes('\r')) {
    return value;
  }

  return value
    .replace(/\\r\\n/g, '\n')
    .replace(/\\n/g, '\n')
    .replace(/\\r/g, '\n')
    .replace(/\\t/g, '\t');
}
