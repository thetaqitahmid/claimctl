import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useRealtime } from './useRealtime';

const control = vi.hoisted(() => ({
  user: { id: 'first' } as { id: string } | null,
  dispatch: vi.fn(),
  notify: vi.fn(),
}));

vi.mock('../store/store', () => ({
  useAppDispatch: () => control.dispatch,
  useAppSelector: () => control.user,
}));
vi.mock('../store/api/reservations', () => ({
  reservationsApiSlice: { util: { invalidateTags: (tags: string[]) => ({ type: 'reservations', tags }) } },
}));
vi.mock('../store/api/resources', () => ({
  resourcesApiSlice: { util: { invalidateTags: (tags: string[]) => ({ type: 'resources', tags }) } },
}));
vi.mock('./useNotification', () => ({
  useNotificationContext: () => ({ showNotification: control.notify }),
}));

class TestEventSource {
  static instances: TestEventSource[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  close = vi.fn();

  constructor() {
    TestEventSource.instances.push(this);
  }
}

describe('useRealtime replica resync', () => {
  beforeEach(() => {
    control.user = { id: 'first' };
    control.dispatch.mockClear();
    control.notify.mockClear();
    TestEventSource.instances = [];
    vi.stubGlobal('EventSource', TestEventSource);
    vi.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('refetches on open, resync and scoped invalidation without fake notifications', () => {
    const { unmount } = renderHook(useRealtime);
    const source = TestEventSource.instances[0];
    act(() => source.onopen?.());
    for (const type of ['resync', 'maintenance_change', 'reservation_update']) {
      act(() => source.onmessage?.({ data: JSON.stringify({ type, payload: { resource_id: 'uuid' } }) }));
    }
    expect(control.dispatch).toHaveBeenCalledTimes(8);
    expect(control.notify).not.toHaveBeenCalled();
    unmount();
    expect(source.close).toHaveBeenCalledOnce();
  });

  it('preserves native reconnect and closes the old principal stream', () => {
    const { rerender, unmount } = renderHook(useRealtime);
    const first = TestEventSource.instances[0];
    act(() => first.onerror?.());
    expect(first.close).not.toHaveBeenCalled();
    expect(control.dispatch).toHaveBeenCalledTimes(2);
    control.user = { id: 'second' };
    rerender();
    expect(first.close).toHaveBeenCalledOnce();
    expect(TestEventSource.instances).toHaveLength(2);
    const dispatchCount = control.dispatch.mock.calls.length;
    act(() => first.onmessage?.({ data: JSON.stringify({ type: 'reservation_update', payload: { action: 'created', resource_name: 'old principal' } }) }));
    expect(control.dispatch).toHaveBeenCalledTimes(dispatchCount);
    expect(control.notify).not.toHaveBeenCalled();
    control.user = null;
    rerender();
    expect(TestEventSource.instances[1].close).toHaveBeenCalledOnce();
    expect(TestEventSource.instances).toHaveLength(2);
    unmount();
  });
});
