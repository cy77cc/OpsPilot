import { describe, expect, it } from 'vitest';
import { RunReconnectController } from '../reconnectController';

describe('reconnectController', () => {
  it('returns retry params for a running pending run', async () => {
    const controller = new RunReconnectController();
    controller.handleMeta({ run_id: 'r1', session_id: 's1', turn: 1 }, { message: 'hi', clientRequestId: 'req-1' });
    controller.handleEventId('evt-1');
    controller.handleRunState({ run_id: 'r1', status: 'running' });

    const next = await controller.nextAttempt();
    expect(next).toEqual({
      message: '',
      sessionId: 's1',
      clientRequestId: 'req-1',
      lastEventId: 'evt-1',
    });
  });
});
