import { describe, expect, it } from 'vitest';

import {
  resolveSessionInboundIds,
  toGuidSessionMap,
  toGuidTagMap,
} from '@/lib/traffic/session-inbound';

describe('session inbound helpers', () => {
  it('maps active tags by guid', () => {
    const map = toGuidTagMap({ 'panel-guid': ['in-awg', 'in-vless'] });
    expect(map.get('panel-guid')?.has('in-awg')).toBe(true);
  });

  it('maps session tags by guid', () => {
    const map = toGuidSessionMap({ g: { 'user@test': 'awg-tag' } });
    expect(map.get('g')?.get('user@test')).toBe('awg-tag');
  });

  it('resolves session inbound from attributed tag', () => {
    const session = toGuidSessionMap({ g: { 'user@test': 'awg-tag' } });
    const ids = resolveSessionInboundIds(
      'user@test',
      [1, 2],
      { 1: { tag: 'awg-tag' }, 2: { tag: 'vless-tag' } },
      new Set(['user@test']),
      session,
      new Map(),
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
      new Map(),
    );
    expect(ids).toEqual([9]);
  });

  it('returns empty when multiple inbounds lack session attribution', () => {
    const active = toGuidTagMap({ g: ['awg-tag', 'vless-tag'] });
    const ids = resolveSessionInboundIds(
      'user@test',
      [1, 2],
      { 1: { tag: 'awg-tag' }, 2: { tag: 'vless-tag' } },
      new Set(['user@test']),
      new Map(),
      active,
    );
    expect(ids).toEqual([]);
  });
});
