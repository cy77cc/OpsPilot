export type NavigateFn = (url: string) => void;

const ALLOWED_PROTOCOLS = new Set(['http:', 'https:']);
const HAS_SCHEME_RE = /^[a-zA-Z][a-zA-Z\d+\-.]*:/;

export function isSafeNavigateUrl(url: string): boolean {
  const raw = url.trim();
  if (!raw) {
    return false;
  }

  // Allow only slash-prefixed relative paths (not protocol-relative URLs).
  if (raw.startsWith('/')) {
    return !raw.startsWith('//');
  }

  // Reject non-slash relative URLs such as "foo".
  if (!HAS_SCHEME_RE.test(raw)) {
    return false;
  }

  try {
    const parsed = new URL(raw, window.location.origin);
    if (!ALLOWED_PROTOCOLS.has(parsed.protocol.toLowerCase())) {
      return false;
    }
    return parsed.origin === window.location.origin;
  } catch {
    return false;
  }
}

export function safeNavigate(url?: string | null, navigate: NavigateFn = (nextUrl) => window.location.assign(nextUrl)): boolean {
  if (typeof url !== 'string') {
    return false;
  }

  const trimmedUrl = url.trim();
  if (!isSafeNavigateUrl(trimmedUrl)) {
    return false;
  }

  navigate(trimmedUrl);
  return true;
}
