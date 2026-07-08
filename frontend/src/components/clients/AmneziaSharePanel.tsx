import { DownloadOutlined } from '@ant-design/icons';
import { Alert, Button, Segmented, Space } from 'antd';
import { useState } from 'react';

import ConfigBlock from '@/components/clients/ConfigBlock';
import { QrPanel } from '@/pages/inbounds/qr';
import './AmneziaSharePanel.css';

interface AmneziaSharePanelProps {
  label: string;
  nativeValue: string;
  vpnUri: string;
  vpnFile?: string;
  fileName?: string;
  qrRemark?: string;
  nativeTabLabel?: string;
  nativeAsLink?: boolean;
}

export default function AmneziaSharePanel({
  label,
  nativeValue,
  vpnUri,
  vpnFile = '',
  fileName = 'config.conf',
  qrRemark = '',
  nativeTabLabel = 'Native config',
  nativeAsLink = false,
}: AmneziaSharePanelProps) {
  const [mode, setMode] = useState<'amnezia' | 'native'>('amnezia');

  const downloadVpnFile = () => {
    if (!vpnFile.trim()) return;
    const blob = new Blob([vpnFile], { type: 'application/json;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = fileName.endsWith('.vpn') ? fileName : fileName.replace(/\.[^.]*$/, '') + '.vpn';
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="amnezia-share-panel">
      <Segmented
        block
        value={mode}
        onChange={(value) => setMode(value as 'amnezia' | 'native')}
        options={[
          { label: 'Amnezia app', value: 'amnezia' },
          { label: nativeTabLabel, value: 'native' },
        ]}
      />
      {mode === 'amnezia' ? (
        <>
          <Alert
            type="info"
            showIcon
            style={{ margin: '12px 0' }}
            message="Scan with Amnezia VPN"
            description="Open Amnezia → add connection → QR code. This imports the full profile in one tap."
          />
          {vpnUri ? (
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <QrPanel
                value={vpnUri}
                remark={qrRemark || 'Amnezia VPN'}
                size={280}
              />
              <Button
                icon={<DownloadOutlined />}
                onClick={downloadVpnFile}
                disabled={!vpnFile.trim()}
              >
                Download `.vpn` for Amnezia
              </Button>
            </Space>
          ) : (
            <Alert
              type="warning"
              showIcon
              message="Amnezia import link unavailable"
              description="Required client keys are missing in the panel database. Regenerate keys for this client or create it through the panel."
            />
          )}
        </>
      ) : nativeAsLink ? (
        <div style={{ marginTop: 12 }}>
          <QrPanel
            value={nativeValue}
            remark={qrRemark || label}
            showQr={!!nativeValue}
          />
        </div>
      ) : (
        <div style={{ marginTop: 12 }}>
          <ConfigBlock
            label={label}
            text={nativeValue}
            fileName={fileName}
            qrRemark={qrRemark}
            showQr={!!nativeValue}
            tagColor="cyan"
            defaultOpen
          />
        </div>
      )}
    </div>
  );
}
