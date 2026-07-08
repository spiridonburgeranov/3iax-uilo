import { useEffect, useState } from 'react';
import type { ClientRecord, InboundOption } from '@/hooks/useClients';
import {
  clientTunnelConfigLabel,
  findTunnelInbounds,
  resolveTunnelClientConfig,
} from './wireguardConfig';
import { fetchClientVpnUri } from '@/lib/amnezia/share';

export interface TunnelConfigItem {
  id: number;
  label: string;
  text: string;
  vpnUri: string;
  protocol: 'wireguard' | 'amneziawg';
}

export function useTunnelClientConfigs(
  open: boolean,
  client: ClientRecord | null,
  inboundsById: Record<number, InboundOption>,
  publicHost = '',
) {
  const [tunnelConfigs, setTunnelConfigs] = useState<TunnelConfigItem[]>([]);
  const [tunnelConfigsLoading, setTunnelConfigsLoading] = useState(false);

  useEffect(() => {
    if (!open || !client) {
      setTunnelConfigs([]);
      return;
    }
    let cancelled = false;
    setTunnelConfigsLoading(true);
    void (async () => {
      const inbounds = findTunnelInbounds(client, inboundsById);
      const loaded = await Promise.all(inbounds.map(async (inbound) => {
        const text = await resolveTunnelClientConfig(
          client,
          inbound,
          window.location.hostname,
          publicHost,
        );
        const vpnUri = await fetchClientVpnUri(client, inbound, window.location.hostname, publicHost);
        return {
          id: inbound.id,
          label: clientTunnelConfigLabel(inbound),
          text,
          vpnUri,
          protocol: inbound.protocol as 'wireguard' | 'amneziawg',
        };
      }));
      if (!cancelled) {
        setTunnelConfigs(loaded.filter((item) => item.text.length > 0 || item.vpnUri.length > 0));
        setTunnelConfigsLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [open, client, inboundsById, publicHost]);

  return { tunnelConfigs, tunnelConfigsLoading };
}
