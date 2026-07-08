import { preferPublicHost, resolveShareHost } from '@/lib/xray/inbound-link';
import { HttpUtil } from '@/utils';
import type { ClientRecord, InboundOption } from '@/hooks/useClients';

export const AMNEZIA_VPN_URI_PROTOCOLS = new Set([
  'vmess',
  'vless',
  'trojan',
  'shadowsocks',
  'hysteria',
  'wireguard',
  'amneziawg',
]);

export function supportsAmneziaVpnUri(protocol: string | undefined): boolean {
  return !!protocol && AMNEZIA_VPN_URI_PROTOCOLS.has(protocol);
}

export function amneziaNativeTabLabel(protocol: string | undefined): string {
  switch (protocol) {
    case 'wireguard':
    case 'amneziawg':
      return 'WireGuard .conf';
    case 'vmess':
    case 'vless':
    case 'trojan':
    case 'shadowsocks':
    case 'hysteria':
      return 'Share link';
    default:
      return 'Native config';
  }
}

export async function fetchClientVpnUri(
  client: ClientRecord,
  inbound: InboundOption,
  host = window.location.hostname,
  publicHost = '',
): Promise<string> {
  const endpointHost = resolveShareHost(inbound, inbound.nodeAddress ?? '', preferPublicHost(host, publicHost));
  const endpoint = `${endpointHost}:${inbound.port || ''}`;
  const path = inbound.protocol === 'amneziawg'
    ? `/panel/api/awg/client/${inbound.id}/${encodeURIComponent(client.email)}/vpnuri`
    : `/panel/api/clients/inbound/${inbound.id}/${encodeURIComponent(client.email)}/vpnuri`;
  const msg = await HttpUtil.get<string>(path, { endpoint }, { silent: true });
  if (msg.success && typeof msg.obj === 'string' && msg.obj.trim()) {
    return msg.obj;
  }
  return '';
}
