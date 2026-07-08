import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Collapse, Modal, Spin, Tag } from 'antd';
import { HttpUtil } from '@/utils';
import { isPostQuantumLink } from '@/lib/xray/inbound-link';
import { LinkTags, linkMetaText, parseLinkParts } from '@/lib/xray/link-label';
import { QrPanel } from '@/pages/inbounds/qr';
import AmneziaSharePanel from '@/components/clients/AmneziaSharePanel';
import { amneziaNativeTabLabel } from '@/lib/amnezia/share';
import { useAmneziaClientShares } from '@/hooks/useAmneziaClientShares';
import type { ClientRecord, InboundOption } from '@/hooks/useClients';
import { useTunnelClientConfigs } from './useTunnelClientConfigs';

interface SubSettings {
  enable: boolean;
  subURI: string;
  subJsonURI: string;
  subJsonEnable: boolean;
  publicHost?: string;
}

interface ClientQrModalProps {
  open: boolean;
  client: ClientRecord | null;
  inboundsById: Record<number, InboundOption>;
  subSettings?: SubSettings;
  onOpenChange: (open: boolean) => void;
}

interface ApiMsg<T = unknown> {
  success?: boolean;
  obj?: T;
}

const DEFAULT_SUB: SubSettings = { enable: false, subURI: '', subJsonURI: '', subJsonEnable: false, publicHost: '' };

export default function ClientQrModal({
  open,
  client,
  inboundsById,
  subSettings = DEFAULT_SUB,
  onOpenChange,
}: ClientQrModalProps) {
  const { t } = useTranslation();
  const [links, setLinks] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);

  const subLink = useMemo(() => {
    if (!client?.subId || !subSettings?.enable || !subSettings?.subURI) return '';
    return subSettings.subURI + client.subId;
  }, [client?.subId, subSettings?.enable, subSettings?.subURI]);

  const subJsonLink = useMemo(() => {
    if (!client?.subId || !subSettings?.enable) return '';
    if (!subSettings?.subJsonEnable || !subSettings?.subJsonURI) return '';
    return subSettings.subJsonURI + client.subId;
  }, [client?.subId, subSettings?.enable, subSettings?.subJsonEnable, subSettings?.subJsonURI]);

  const { tunnelConfigs, tunnelConfigsLoading } = useTunnelClientConfigs(
    open,
    client,
    inboundsById,
    subSettings?.publicHost ?? '',
  );
  const { amneziaShares, amneziaSharesLoading } = useAmneziaClientShares(
    open,
    client,
    inboundsById,
    subSettings?.publicHost ?? '',
  );

  const hasAnything = !!subLink || !!subJsonLink || tunnelConfigs.length > 0 || links.length > 0 || amneziaShares.length > 0;

  useEffect(() => {
    if (!open || !client?.subId) {
      setLinks([]);
      return;
    }
    let cancelled = false;
    setLoading(true);
    (async () => {
      try {
        const msg = await HttpUtil.get(
          `/panel/api/clients/subLinks/${encodeURIComponent(client.subId!)}`,
        ) as ApiMsg<string[]>;
        if (!cancelled) {
          setLinks(msg?.success && Array.isArray(msg.obj) ? msg.obj : []);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [open, client?.subId]);

  const [activeKey, setActiveKey] = useState<string[]>([]);

  const items = useMemo(() => {
    const out: { key: string; label: React.ReactNode; children: React.ReactNode }[] = [];
    if (subLink) {
      out.push({
        key: 'sub',
        label: t('subscription.title'),
        children: <QrPanel value={subLink} remark={`${client?.email || ''} — ${t('subscription.title')}`} />,
      });
    }
    if (subJsonLink) {
      out.push({
        key: 'subJson',
        label: `${t('subscription.title')} (JSON)`,
        children: <QrPanel value={subJsonLink} remark={`${client?.email || ''} — JSON`} />,
      });
    }
    links.forEach((link, idx) => {
      const parts = parseLinkParts(link);
      const meta = parts ? linkMetaText(parts) : '';
      const label: React.ReactNode = parts ? (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
          <LinkTags parts={parts} />
          {meta && <span style={{ opacity: 0.6, fontSize: 12 }}>({meta})</span>}
        </span>
      ) : `${t('pages.clients.link')} ${idx + 1}`;
      out.push({
        key: `l${idx}`,
        label,
        children: (
          <QrPanel
            value={link}
            remark={parts?.remark || `${client?.email || ''} #${idx + 1}`}
            showQr={!isPostQuantumLink(link)}
          />
        ),
      });
    });
    tunnelConfigs.forEach((cfg) => {
      out.push({
        key: `tunnel-config-${cfg.id}`,
        label: <Tag color={cfg.protocol === 'amneziawg' ? 'cyan' : 'gold'} style={{ margin: 0 }}>{cfg.label}</Tag>,
        children: (
          <AmneziaSharePanel
            label={cfg.label}
            nativeValue={cfg.text}
            vpnUri={cfg.vpnUri}
            fileName={`${client?.email || 'peer'}-${cfg.label.toLowerCase().replace(/\s+/g, '-')}.conf`}
            qrRemark={client?.email || 'peer'}
            nativeTabLabel={amneziaNativeTabLabel(cfg.protocol)}
          />
        ),
      });
    });
    amneziaShares.forEach((share) => {
      if (share.protocol === 'wireguard' || share.protocol === 'amneziawg') return;
      out.push({
        key: `amnezia-share-${share.inboundId}`,
        label: <Tag color="purple" style={{ margin: 0 }}>{share.label}</Tag>,
        children: (
          <AmneziaSharePanel
            label={share.label}
            nativeValue={share.nativeValue}
            vpnUri={share.vpnUri}
            qrRemark={client?.email || share.label}
            nativeTabLabel={amneziaNativeTabLabel(share.protocol)}
            nativeAsLink={share.nativeAsLink}
          />
        ),
      });
    });
    return out;
  }, [subLink, subJsonLink, tunnelConfigs, amneziaShares, links, client?.email, t]);

  useEffect(() => {
    if (!open) {
      setActiveKey([]);
      return;
    }
    setActiveKey(items.length > 0 ? [items[0].key] : []);
  }, [open, items]);

  return (
    <Modal
      open={open}
      title={client ? `${t('qrCode')} — ${client.email}` : t('qrCode')}
      footer={null}
      width={600}
      centered
      onCancel={() => onOpenChange(false)}
    >
      <Spin spinning={loading || tunnelConfigsLoading || amneziaSharesLoading}>
        {!hasAnything && !loading && !tunnelConfigsLoading && !amneziaSharesLoading && (
          <div style={{ padding: 24, textAlign: 'center', opacity: 0.6 }}>
            {!client?.subId && tunnelConfigs.length === 0
              ? t('pages.clients.noSubId')
              : t('pages.clients.noLinks')}
          </div>
        )}
        {hasAnything && (
          <Collapse
            activeKey={activeKey}
            onChange={(keys) => setActiveKey(typeof keys === 'string' ? [keys] : (keys as string[]))}
            items={items}
          />
        )}
      </Spin>
    </Modal>
  );
}
