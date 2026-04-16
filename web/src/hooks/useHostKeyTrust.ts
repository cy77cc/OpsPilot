import { useCallback, useState } from 'react';
import { Api } from '../api';
import type { HostKeyTrustErrorData, HostKeyTrustPayload } from '../api/modules/hosts';

type ParsedTrustErrorType = HostKeyTrustErrorData['errorType'];

function inferTrustErrorType(rawType: unknown, message: unknown): ParsedTrustErrorType | null {
  const type = String(rawType || '').trim();
  if (type === 'ssh_host_key_unknown' || type === 'ssh_host_key_mismatch' || type === 'ssh_host_key_revoked') {
    return type;
  }
  const lowered = String(message || '').toLowerCase();
  if (lowered.includes('revoked')) {
    return 'ssh_host_key_revoked';
  }
  if (lowered.includes('mismatch')) {
    return 'ssh_host_key_mismatch';
  }
  if (lowered.includes('unknown') || lowered.includes('host key')) {
    return 'ssh_host_key_unknown';
  }
  return null;
}

function toHostKeyTrustPayload(raw: any): HostKeyTrustPayload | null {
  const payload: HostKeyTrustPayload = {
    host: String(raw?.host || '').trim(),
    port: Number(raw?.port || 0),
    algorithm: String(raw?.algorithm || '').trim(),
    fingerprintSha256: String(raw?.fingerprint_sha256 || raw?.fingerprintSha256 || '').trim(),
    publicKey: String(raw?.public_key || raw?.publicKey || '').trim(),
    knownHostsPath: raw?.known_hosts_path || raw?.knownHostsPath || undefined,
    trustedFingerprints: Array.isArray(raw?.trusted_fingerprints)
      ? raw.trusted_fingerprints.map((x: unknown) => String(x).trim()).filter(Boolean)
      : Array.isArray(raw?.trustedFingerprints)
        ? raw.trustedFingerprints.map((x: unknown) => String(x).trim()).filter(Boolean)
        : undefined,
  };
  if (!payload.host || payload.port <= 0 || !payload.fingerprintSha256 || !payload.publicKey) {
    return null;
  }
  return payload;
}

export function parseHostKeyTrustError(error: unknown): HostKeyTrustErrorData | null {
  const data = (error as any)?.details || (error as any)?.response?.data?.data;
  const message = (error as any)?.message;
  const hostKey = toHostKeyTrustPayload(data?.host_key || data?.hostKey);
  const errorType = inferTrustErrorType(data?.error_type || data?.errorType, message || data?.message);
  if (!hostKey || !errorType) {
    return null;
  }
  return {
    errorType,
    hostKey,
    probeToken: typeof data?.probe_token === 'string' ? data.probe_token.trim() || undefined : undefined,
  };
}

export function useHostKeyTrust(hostId: string) {
  const [pendingTrust, setPendingTrust] = useState<HostKeyTrustErrorData | null>(null);
  const [confirming, setConfirming] = useState(false);

  const runWithTrustRetry = useCallback(async <T,>(operation: () => Promise<T>): Promise<T> => {
    try {
      return await operation();
    } catch (error) {
      const trustError = parseHostKeyTrustError(error);
      if (!trustError) {
        throw error;
      }
      setPendingTrust(trustError);
      throw error;
    }
  }, []);

  const confirmTrustAndRetry = useCallback(async (retry: () => Promise<void>) => {
    if (!pendingTrust || !hostId) {
      return;
    }
    setConfirming(true);
    try {
      await Api.hosts.trustHostKey(hostId, {
        ...pendingTrust.hostKey,
        probeToken: pendingTrust.probeToken,
        replaceExisting: pendingTrust.errorType !== 'ssh_host_key_unknown',
      });
      setPendingTrust(null);
      await retry();
    } finally {
      setConfirming(false);
    }
  }, [hostId, pendingTrust]);

  return {
    pendingTrust,
    setPendingTrust,
    confirming,
    runWithTrustRetry,
    confirmTrustAndRetry,
  };
}
