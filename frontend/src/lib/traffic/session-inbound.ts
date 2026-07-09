export function toGuidTagMap(data: Record<string, string[]>): Map<string, Set<string>> {
  const map = new Map<string, Set<string>>();
  for (const [key, tags] of Object.entries(data)) {
    if (!Array.isArray(tags)) continue;
    map.set(key, new Set(tags.filter((tag) => !!tag && tag.trim().length > 0)));
  }
  return map;
}

export function resolveSessionInboundIds(
  email: string,
  inboundIds: number[],
  inboundsById: Record<number, { tag?: string } | undefined>,
  onlineEmails: Set<string>,
  activeByGuid: Map<string, Set<string>>,
): number[] {
  if (!email || !onlineEmails.has(email) || inboundIds.length === 0) return [];
  const matched: number[] = [];
  for (const id of inboundIds) {
    const tag = inboundsById[id]?.tag?.trim();
    if (!tag) continue;
    for (const tags of activeByGuid.values()) {
      if (tags.has(tag)) {
        matched.push(id);
        break;
      }
    }
  }
  if (matched.length > 0) return matched;
  if (inboundIds.length === 1) return inboundIds;
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
