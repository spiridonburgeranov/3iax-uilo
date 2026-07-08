import { formatInboundLabel } from '@/lib/inbounds/label';
import { preferPublicHost, resolveShareHost } from '@/lib/xray/inbound-link';
import { HttpUtil } from '@/utils';
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


function awgOptionalNum(value: number | undefined) {
  return typeof value === 'number' && !Number.isNaN(value) ? value : undefined;
}

function awgText(value: string | undefined) {
  return typeof value === 'string' ? value.trim() : '';
}

function clientAllowedIPs(client: ClientRecord): string {
  const value = client.allowedIPs as string | string[] | undefined;
  if (Array.isArray(value)) {
    const parts = value.map((v) => String(v).trim()).filter(Boolean);
    if (parts.length) return parts.join(', ');
  }
  if (typeof value === 'string' && value.trim()) return value.trim();
  return '10.66.66.2/32';
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

export function buildAmneziawgClientConfig(
  client: ClientRecord,
  inbound: InboundOption | undefined,
  host = window.location.hostname,
  publicHost = '',
): string {
  const endpointHost = resolveShareHost(inbound ?? {}, inbound?.nodeAddress ?? '', preferPublicHost(host, publicHost));
  const address = clientAllowedIPs(client);
  const endpoint = `${endpointHost}:${inbound?.port || ''}`;
  const inboundName = inbound ? formatInboundLabel(inbound.tag, inbound.remark) : '';
  const remark = [inboundName, client.email, client.comment].filter(Boolean).join(' - ');
  const lines = [
    '[Interface]',
    `PrivateKey = ${client.privateKey || ''}`,
    `Address = ${address}`,
    `DNS = ${inbound?.wgDns || '1.1.1.1,2606:4700:4700::1111'}`,
  ];
  if (inbound?.wgMtu && inbound.wgMtu > 0) lines.push(`MTU = ${inbound.wgMtu}`);
  const obfuscation = [
    ['Jc', awgOptionalNum(inbound?.awgJc)],
    ['Jmin', awgOptionalNum(inbound?.awgJmin)],
    ['Jmax', awgOptionalNum(inbound?.awgJmax)],
    ['S1', awgOptionalNum(inbound?.awgS1)],
    ['S2', awgOptionalNum(inbound?.awgS2)],
    ['S3', awgOptionalNum(inbound?.awgS3)],
    ['S4', awgOptionalNum(inbound?.awgS4)],
  ] as const;
  for (const [label, value] of obfuscation) {
    if (value !== undefined) lines.push(`${label} = ${value}`);
  }
  for (const [label, value] of [
    ['H1', inbound?.awgH1],
    ['H2', inbound?.awgH2],
    ['H3', inbound?.awgH3],
    ['H4', inbound?.awgH4],
    ['I1', inbound?.awgI1],
    ['I2', inbound?.awgI2],
    ['I3', inbound?.awgI3],
    ['I4', inbound?.awgI4],
    ['I5', inbound?.awgI5],
  ] as const) {
    const text = awgText(value);
    if (text) lines.push(`${label} = ${text}`);
  }
  lines.push('');
  if (remark) lines.push(`# ${remark}`);
  lines.push('[Peer]', `PublicKey = ${inbound?.wgPublicKey || ''}`);
  if (client.preSharedKey) lines.push(`PresharedKey = ${client.preSharedKey}`);
  lines.push('AllowedIPs = 0.0.0.0/0, ::/0', `Endpoint = ${endpoint}`);
  const keepAlive = client.keepAlive && client.keepAlive > 0 ? client.keepAlive : 25;
  lines.push(`PersistentKeepalive = ${keepAlive}`);
  return lines.join('\n');
}

export async function fetchAmneziawgClientConfig(
  client: ClientRecord,
  inbound: InboundOption,
  host = window.location.hostname,
  publicHost = '',
): Promise<string> {
  const endpointHost = resolveShareHost(inbound, inbound.nodeAddress ?? '', preferPublicHost(host, publicHost));
  const endpoint = `${endpointHost}:${inbound.port || ''}`;
  const msg = await HttpUtil.get<string>(
    `/panel/api/awg/client/${inbound.id}/${encodeURIComponent(client.email)}/config`,
    { endpoint },
    { silent: true },
  );
  if (msg.success && typeof msg.obj === 'string' && msg.obj.trim()) {
    return msg.obj;
  }
  return buildAmneziawgClientConfig(client, inbound, host, publicHost);
}

export async function fetchAmneziawgVpnUri(
  client: ClientRecord,
  inbound: InboundOption,
  host = window.location.hostname,
  publicHost = '',
): Promise<string> {
  const endpointHost = resolveShareHost(inbound, inbound.nodeAddress ?? '', preferPublicHost(host, publicHost));
  const endpoint = `${endpointHost}:${inbound.port || ''}`;
  const msg = await HttpUtil.get<string>(
    `/panel/api/awg/client/${inbound.id}/${encodeURIComponent(client.email)}/vpnuri`,
    { endpoint },
    { silent: true },
  );
  if (msg.success && typeof msg.obj === 'string' && msg.obj.trim()) {
    return msg.obj;
  }
  return '';
}

export async function resolveTunnelClientConfig(
  client: ClientRecord,
  inbound: InboundOption | undefined,
  host = window.location.hostname,
  publicHost = '',
): Promise<string> {
  if (!inbound) return '';
  if (inbound.protocol === 'amneziawg') {
    return fetchAmneziawgClientConfig(client, inbound, host, publicHost);
  }
  return buildWireguardClientConfig(client, inbound, host, publicHost);
}

export function buildClientTunnelConfig(
  client: ClientRecord,
  inbound: InboundOption | undefined,
  host = window.location.hostname,
  publicHost = '',
): string {
  if (inbound?.protocol === 'amneziawg') {
    return buildAmneziawgClientConfig(client, inbound, host, publicHost);
  }
  return buildWireguardClientConfig(client, inbound, host, publicHost);
}

export function clientTunnelConfigLabel(inbound: InboundOption | undefined): string {
  return inbound?.protocol === 'amneziawg' ? 'AmneziaWG config' : 'WireGuard config';
}
