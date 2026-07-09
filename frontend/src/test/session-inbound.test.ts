import { describe, expect, it } from 'vitest';

import { resolveSessionInboundIds, toGuidTagMap } from '@/lib/traffic/session-inbound';

describe('session inbound helpers', () => {
  it('maps active tags by guid', () => {
    const map = toGuidTagMap({ 'panel-guid': ['in-awg', 'in-vless'] });
    expect(map.get('panel-guid')?.has('in-awg')).toBe(true);
  });

  it('resolves session inbound from active tag', () => {
    const active = toGuidTagMap({ g: ['awg-tag'] });
    const ids = resolveSessionInboundIds(
      'user@test',
      [1, 2],
      { 1: { tag: 'awg-tag' }, 2: { tag: 'vless-tag' } },
      new Set(['user@test']),
      active,
    );
    expect(ids).toEqual([1]);
  });

  it('falls back to sole attached inbound when online', () => {
    const ids = resolveSessionInboundIds(
      'user@test',
      [9],
      { 9: { tag: 'only' } },
      new Set(['user@test']),
      new Map(),
    );
    expect(ids).toEqual([9]);
  });
});
