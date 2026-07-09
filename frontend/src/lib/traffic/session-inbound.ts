export function toGuidTagMap(data: Record<string, string[]>): Map<string, Set<string>> {
  const map = new Map<string, Set<string>>();
  for (const [key, tags] of Object.entries(data)) {
    if (!Array.isArray(tags)) continue;
    map.set(key, new Set(tags.filter((tag) => !!tag && tag.trim().length > 0)));
  }
  return map;
}

export type InboundSessionMeta = {
  tag?: string;
  originNodeGuid?: string;
  nodeId?: number | null;
};

export function inboundGuid(meta: InboundSessionMeta): string {
  if (meta.originNodeGuid) return meta.originNodeGuid;
  if (meta.nodeId != null) return `node:${meta.nodeId}`;
  return '';
}

function lookupSessionTag(
  email: string,
  meta: InboundSessionMeta,
  sessionByGuid: Map<string, Map<string, string>>,
): string | undefined {
  const guid = inboundGuid(meta);
  if (guid) {
    return sessionByGuid.get(guid)?.get(email);
  }
  for (const byEmail of sessionByGuid.values()) {
    const tag = byEmail.get(email);
    if (tag) return tag;
  }
  return undefined;
}

export function toGuidSessionMap(
  data: Record<string, Record<string, string>> | undefined | null,
): Map<string, Map<string, string>> {
  const map = new Map<string, Map<string, string>>();
  if (!data || typeof data !== 'object') return map;
  for (const [guid, tags] of Object.entries(data)) {
    if (!tags || typeof tags !== 'object') continue;
    const inner = new Map<string, string>();
    for (const [email, tag] of Object.entries(tags)) {
      if (email && tag) inner.set(email, tag);
    }
    if (inner.size > 0) map.set(guid, inner);
  }
  return map;
}

export function resolveSessionInboundIds(
  email: string,
  inboundIds: number[],
  inboundsById: Record<number, InboundSessionMeta | undefined>,
  onlineEmails: Set<string>,
  sessionByGuid: Map<string, Map<string, string>>,
  activeByGuid: Map<string, Set<string>>,
): number[] {
  if (!email || !onlineEmails.has(email) || inboundIds.length === 0) return [];

  for (const id of inboundIds) {
    const meta = inboundsById[id];
    const tag = meta?.tag?.trim();
    if (!tag) continue;
    const sessionTag = lookupSessionTag(email, meta ?? {}, sessionByGuid);
    if (sessionTag && sessionTag === tag) return [id];
  }

  if (inboundIds.length === 1) return inboundIds;

  const activeMatches: number[] = [];
  for (const id of inboundIds) {
    const meta = inboundsById[id];
    const tag = meta?.tag?.trim();
    if (!tag) continue;
    const guid = inboundGuid(meta ?? {});
    const activeForNode = activeByGuid.get(guid);
    if (activeForNode?.has(tag)) activeMatches.push(id);
  }
  if (activeMatches.length === 1) return activeMatches;
  return [];
}

export function protocolLabel(protocol: string | undefined): string {
  const p = (protocol || '').toLowerCase();
  switch (p) {
    case 'amneziawg':
      return 'AmneziaWG';
    case 'wireguard':
      return 'WireGuard';
    case 'vless':
      return 'VLESS';
    case 'vmess':
      return 'VMess';
    case 'trojan':
      return 'Trojan';
    case 'shadowsocks':
      return 'Shadowsocks';
    case 'hysteria':
      return 'Hysteria';
    case 'mtproto':
      return 'MTProto';
    default:
      return p || '—';
  }
}
