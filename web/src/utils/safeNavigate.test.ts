import { describe, expect, it, vi } from 'vitest';
import { safeNavigate } from './safeNavigate';

describe('safeNavigate', () => {
  it('navigates for a relative URL', () => {
    const navigate = vi.fn();

    expect(safeNavigate('/dashboard', navigate)).toBe(true);
    expect(navigate).toHaveBeenCalledWith('/dashboard');
  });

  it('rejects javascript URLs', () => {
    const navigate = vi.fn();

    expect(safeNavigate('javascript:alert(1)', navigate)).toBe(false);
    expect(navigate).not.toHaveBeenCalled();
  });

  it('rejects javascript URLs with spaces and casing tricks', () => {
    const navigate = vi.fn();

    expect(safeNavigate('  JavaScript:alert(1)', navigate)).toBe(false);
    expect(navigate).not.toHaveBeenCalled();
  });

  it('rejects cross-origin absolute URLs', () => {
    const navigate = vi.fn();

    expect(safeNavigate('https://example.com/redirect', navigate)).toBe(false);
    expect(navigate).not.toHaveBeenCalled();
  });

  it('rejects protocol-relative URLs', () => {
    const navigate = vi.fn();

    expect(safeNavigate('//evil.com', navigate)).toBe(false);
    expect(navigate).not.toHaveBeenCalled();
  });

  it('rejects non-slash relative URLs', () => {
    const navigate = vi.fn();

    expect(safeNavigate('foo', navigate)).toBe(false);
    expect(navigate).not.toHaveBeenCalled();
  });

  it('allows same-origin absolute URLs', () => {
    const navigate = vi.fn();
    const sameOriginUrl = `${window.location.origin}/monitor?alert_id=alert-001`;

    expect(safeNavigate(sameOriginUrl, navigate)).toBe(true);
    expect(navigate).toHaveBeenCalledWith(sameOriginUrl);
  });
});
