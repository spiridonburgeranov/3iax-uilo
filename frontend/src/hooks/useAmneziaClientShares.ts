import { useEffect, useState } from 'react';
import { inboundFromDb, type DbInboundLike } from '@/lib/xray/inbound-from-db';
import { genAllLinks } from '@/lib/xray/inbound-link';
import { fetchClientVpnFile, fetchClientVpnUri, supportsAmneziaVpnUri } from '@/lib/amnezia/share';
import type { ClientRecord, InboundOption } from '@/hooks/useClients';

export interface AmneziaShareItem {
  inboundId: number;
  protocol: string;
  label: string;
  nativeValue: string;
  vpnUri: string;
  vpnFile: string;
  nativeAsLink: boolean;
}

export function useAmneziaClientShares(
  open: boolean,
  client: ClientRecord | null,
  inboundsById: Record<number, InboundOption>,
  publicHost = '',
) {
  const [items, setItems] = useState<AmneziaShareItem[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open || !client) {
      setItems([]);
      return;
    }
    let cancelled = false;
    setLoading(true);
    void (async () => {
      const inbounds = (client.inboundIds || [])
        .map((id) => inboundsById[id])
        .filter((ib): ib is InboundOption => !!ib
          && supportsAmneziaVpnUri(ib.protocol)
          && ib.protocol !== 'wireguard'
          && ib.protocol !== 'amneziawg');
      const loaded = await Promise.all(inbounds.map(async (inbound) => {
        const vpnUri = await fetchClientVpnUri(client, inbound, window.location.hostname, publicHost);
        const vpnFile = await fetchClientVpnFile(client, inbound, window.location.hostname, publicHost);
        let nativeValue = '';
        let nativeAsLink = false;
        if (inbound.protocol === 'wireguard' || inbound.protocol === 'amneziawg') {
          nativeAsLink = false;
        } else {
          nativeAsLink = true;
          const dbInbound = inbound as unknown as DbInboundLike & { remark?: string };
          const links = genAllLinks({
            inbound: inboundFromDb(dbInbound),
            remark: inbound.remark || '',
            client: {
              id: client.uuid || client.email,
              email: client.email,
              password: client.password,
              flow: client.flow as '' | 'xtls-rprx-vision' | 'xtls-rprx-vision-udp443' | undefined,
              security: client.security as 'auto' | 'none' | 'aes-128-gcm' | 'chacha20-poly1305' | 'zero' | undefined,
              auth: client.auth,
              secret: client.secret,
              subId: client.subId,
            },
            hostOverride: inbound.nodeAddress ?? '',
            fallbackHostname: publicHost,
          });
          nativeValue = links[0]?.link || '';
        }
        const protocol = inbound.protocol || '';
        return {
          inboundId: inbound.id,
          protocol,
          label: inbound.remark || inbound.tag || protocol,
          nativeValue,
          vpnUri,
          vpnFile,
          nativeAsLink,
        };
      }));
      if (!cancelled) {
        setItems(loaded.filter((item) => item.vpnUri || item.vpnFile || item.nativeValue));
        setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [open, client, inboundsById, publicHost]);

  return { amneziaShares: items, amneziaSharesLoading: loading };
}
