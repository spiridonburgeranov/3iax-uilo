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
  return findTunnelInbounds(client, inboundsById).find((ib) => ib.protocol === 'wireguard');
}

export function findTunnelInbounds(
  client: ClientRecord | null | undefined,
  inboundsById: Record<number, InboundOption>,
): InboundOption[] {
  return (client?.inboundIds || [])
    .map((id) => inboundsById[id])
    .filter((ib): ib is InboundOption => ib?.protocol === 'wireguard' || ib?.protocol === 'amneziawg');
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
  const inboundName = inbound ? formatInboundLabel(inbound.tag, inbound.remark) : '';
  const remark = [inboundName, client.email, client.comment].filter(Boolean).join(' - ');
  const lines = [
    '[Interface]',
    `PrivateKey = ${client.privateKey || client.password || ''}`,
    `Address = ${address}`,
    `DNS = ${inbound?.wgDns || '1.1.1.1, 1.0.0.1'}`,
  ];
  if (inbound?.wgMtu && inbound.wgMtu > 0) lines.push(`MTU = ${inbound.wgMtu}`);
  lines.push('');
  if (remark) lines.push(`# ${remark}`);
  lines.push('[Peer]', `PublicKey = ${inbound?.wgPublicKey || ''}`);
  if (client.preSharedKey) lines.push(`PresharedKey = ${client.preSharedKey}`);
  lines.push('AllowedIPs = 0.0.0.0/0, ::/0', `Endpoint = ${endpoint}`);
  if (client.keepAlive && client.keepAlive > 0) lines.push(`PersistentKeepalive = ${client.keepAlive}`);
  return lines.join('\n');
}

export function buildAmneziaClientConfig(
  client: ClientRecord,
  inbound: InboundOption | undefined,
  host = window.location.hostname,
  publicHost = '',
): string {
  const endpointHost = resolveShareHost(inbound ?? {}, inbound?.nodeAddress ?? '', preferPublicHost(host, publicHost));
  const address = client.allowedIPs || '10.66.66.2/32';
  const endpoint = `${endpointHost}:${inbound?.port || ''}`;
  const lines = [
    '[Interface]',
    `PrivateKey = ${client.privateKey || ''}`,
    `Address = ${address}`,
    `DNS = ${inbound?.wgDns || '1.1.1.1,2606:4700:4700::1111'}`,
  ];
  if (inbound?.wgMtu && inbound.wgMtu > 0) lines.push(`MTU = ${inbound.wgMtu}`);
  for (const [key, value] of [
    ['Jc', inbound?.awgJc ?? 4],
    ['Jmin', inbound?.awgJmin ?? 64],
    ['Jmax', inbound?.awgJmax ?? 256],
    ['S1', inbound?.awgS1 ?? 15],
    ['S2', inbound?.awgS2 ?? 25],
    ['S3', inbound?.awgS3 ?? 35],
    ['S4', inbound?.awgS4 ?? 15],
    ['H1', inbound?.awgH1 ?? 5],
    ['H2', inbound?.awgH2 ?? 10],
    ['H3', inbound?.awgH3 ?? 15],
    ['H4', inbound?.awgH4 ?? 20],
  ] as const) {
    if (typeof value === 'number' && value >= 0) lines.push(`${key} = ${value}`);
  }
  for (const [key, value] of [
    ['I1', inbound?.awgI1],
    ['I2', inbound?.awgI2],
    ['I3', inbound?.awgI3],
    ['I4', inbound?.awgI4],
    ['I5', inbound?.awgI5],
  ] as const) {
    if (value) lines.push(`${key} = ${value}`);
  }
  lines.push('', '[Peer]', `PublicKey = ${inbound?.wgPublicKey || ''}`);
  if (client.preSharedKey) lines.push(`PresharedKey = ${client.preSharedKey}`);
  lines.push(`Endpoint = ${endpoint}`, 'AllowedIPs = 0.0.0.0/0, ::/0');
  const keepAlive = client.keepAlive && client.keepAlive > 0 ? client.keepAlive : 25;
  lines.push(`PersistentKeepalive = ${keepAlive}`);
  return lines.join('\n');
}

export function buildClientTunnelConfig(
  client: ClientRecord,
  inbound: InboundOption | undefined,
  host = window.location.hostname,
  publicHost = '',
): string {
  if (inbound?.protocol === 'amneziawg') {
    return buildAmneziaClientConfig(client, inbound, host, publicHost);
  }
  return buildWireguardClientConfig(client, inbound, host, publicHost);
}

export function clientTunnelConfigLabel(inbound: InboundOption | undefined): string {
  return inbound?.protocol === 'amneziawg' ? 'AmneziaWG config' : 'WireGuard config';
}
