import { formatInboundLabel } from '@/lib/inbounds/label';
import { preferPublicHost, resolveShareHost } from '@/lib/xray/inbound-link';
import type { ClientRecord, InboundOption } from '@/hooks/useClients';

export function isWireguardClient(client: ClientRecord | null | undefined): boolean {
  if (!client) return false;
  return !!(client.privateKey || client.publicKey || client.allowedIPs || client.preSharedKey || client.keepAlive);
}

export function findWireguardInbound(
  client: ClientRecord | null | undefined,
  inboundsById: Record<number, InboundOption>,
): InboundOption | undefined {
  return (client?.inboundIds || [])
    .map((id) => inboundsById[id])
    .find((ib) => ib?.protocol === 'wireguard' || ib?.protocol === 'amneziawg');
}

export function buildWireguardClientConfig(
  client: ClientRecord,
  inbound: InboundOption | undefined,
  host = window.location.hostname,
  publicHost = '',
): string {
  const endpointHost = resolveShareHost(inbound ?? {}, inbound?.nodeAddress ?? '', preferPublicHost(host, publicHost));
  const address = client.allowedIPs || '10.0.0.2/32';
  const endpoint = `${endpointHost}:${inbound?.port || ''}`;
  const isAmneziaWG = inbound?.protocol === 'amneziawg';
  const inboundName = inbound ? formatInboundLabel(inbound.tag, inbound.remark) : '';
  const remark = [inboundName, client.email, client.comment].filter(Boolean).join(' - ');
  const lines = [
    '[Interface]',
    `PrivateKey = ${client.privateKey || client.password || ''}`,
    `Address = ${address}`,
    `DNS = ${inbound?.wgDns || (isAmneziaWG ? '1.1.1.1,2606:4700:4700::1111' : '1.1.1.1, 1.0.0.1')}`,
  ];
  if (inbound?.wgMtu && inbound.wgMtu > 0) lines.push(`MTU = ${inbound.wgMtu}`);
  if (isAmneziaWG) {
    const awgParams = [
      ['Jc', inbound.awgJc ?? 4],
      ['Jmin', inbound.awgJmin ?? 50],
      ['Jmax', inbound.awgJmax ?? 1000],
      ['S1', inbound.awgS1 ?? 0],
      ['S2', inbound.awgS2 ?? 0],
      ['H1', inbound.awgH1 ?? 1],
      ['H2', inbound.awgH2 ?? 2],
      ['H3', inbound.awgH3 ?? 3],
      ['H4', inbound.awgH4 ?? 4],
    ] as const;
    for (const [key, value] of awgParams) {
      if (typeof value === 'number' && value >= 0) lines.push(`${key} = ${value}`);
    }
  }
  lines.push('');
  if (remark) lines.push(`# ${remark}`);
  lines.push('[Peer]', `PublicKey = ${inbound?.wgPublicKey || ''}`);
  if (client.preSharedKey) lines.push(`PresharedKey = ${client.preSharedKey}`);
  lines.push(`Endpoint = ${endpoint}`, 'AllowedIPs = 0.0.0.0/0, ::/0');
  const keepAlive = client.keepAlive && client.keepAlive > 0 ? client.keepAlive : (isAmneziaWG ? 25 : 0);
  if (keepAlive > 0) lines.push(`PersistentKeepalive = ${keepAlive}`);
  return lines.join('\n');
}
