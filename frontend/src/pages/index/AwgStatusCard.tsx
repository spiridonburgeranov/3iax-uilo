import { useMemo, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  Col,
  Drawer,
  Empty,
  Row,
  Space,
  Statistic,
  Tag,
  Typography,
} from 'antd';
import {
  ApiOutlined,
  ArrowDownOutlined,
  ArrowUpOutlined,
  PoweroffOutlined,
  ReloadOutlined,
} from '@ant-design/icons';

import type { AwgPeer, Status } from '@/models/status';
import { SizeFormatter } from '@/utils';
import { activateOnKey } from '@/utils/a11y';
import './AwgStatusCard.css';

const { Text } = Typography;

interface AwgStatusCardProps {
  status: Status;
  isMobile: boolean;
  onStartAwg: () => void;
  onStopAwg: () => void;
  onRestartAwg: () => void;
}

const PREVIEW_PEER_LIMIT = 4;

function peerLabel(peer: AwgPeer, index: number) {
  return peer.email || peer.publicKey?.slice(0, 12) || `peer-${index + 1}`;
}

function PeerRow({ peer, index }: { peer: AwgPeer; index: number }) {
  const label = peerLabel(peer, index);
  const iface = peer.interfaceName || peer.inboundRemark;
  return (
    <div className="awg-status-peer-row">
      <div className="awg-status-peer-name">
        <span className={`awg-status-peer-dot ${peer.online ? 'online' : 'offline'}`} aria-hidden />
        <Text ellipsis={{ tooltip: label }}>{label}</Text>
        <Tag color={peer.online ? 'green' : 'default'} style={{ margin: 0 }}>
          {peer.online ? 'online' : 'idle'}
        </Tag>
      </div>
      <div className="awg-status-peer-traffic">
        ↑ {SizeFormatter.sizeFormat(peer.transferTx || 0)}
        {' · '}
        ↓ {SizeFormatter.sizeFormat(peer.transferRx || 0)}
      </div>
      <div className="awg-status-peer-meta">
        {iface ? <span>{iface}</span> : null}
        {peer.endpoint ? <span>{peer.endpoint}</span> : null}
        {(peer.allowedIPs || []).length > 0 ? (
          <span>{(peer.allowedIPs || []).join(', ')}</span>
        ) : null}
      </div>
    </div>
  );
}

function PeerDrawerRow({ peer, index }: { peer: AwgPeer; index: number }) {
  const label = peerLabel(peer, index);
  return (
    <div className="awg-peer-row">
      <div className="awg-peer-main">
        <span>{label}</span>
        <Tag color={peer.online ? 'green' : 'default'}>{peer.online ? 'online' : 'idle'}</Tag>
      </div>
      {(peer.interfaceName || peer.inboundRemark) && (
        <div className="awg-peer-meta">{peer.interfaceName || peer.inboundRemark}</div>
      )}
      <div className="awg-peer-meta">{peer.endpoint || 'no endpoint'}</div>
      <div className="awg-peer-meta">
        ↑ {SizeFormatter.sizeFormat(peer.transferTx || 0)}
        {' / '}
        ↓ {SizeFormatter.sizeFormat(peer.transferRx || 0)}
      </div>
      {peer.latestHandshake ? (
        <div className="awg-peer-meta">
          handshake {new Date(peer.latestHandshake * 1000).toLocaleString()}
        </div>
      ) : (
        <div className="awg-peer-meta">no handshake</div>
      )}
    </div>
  );
}

export default function AwgStatusCard({
  status,
  isMobile,
  onStartAwg,
  onStopAwg,
  onRestartAwg,
}: AwgStatusCardProps) {
  const [drawerOpen, setDrawerOpen] = useState(false);
  const peers = status.awg.peers || [];
  const running = status.awg.running;
  const installed = status.awg.installed;
  const version = installed ? status.awg.version : 'not installed';
  const toggleHandler = running ? onStopAwg : onStartAwg;
  const toggleLabel = running ? 'Stop' : 'Start';

  const interfaceCount = useMemo(() => {
    const names = new Set<string>();
    for (const peer of peers) {
      const name = (peer.interfaceName || peer.inboundRemark || '').trim();
      if (name) names.add(name);
    }
    return names.size;
  }, [peers]);

  const previewPeers = peers.slice(0, PREVIEW_PEER_LIMIT);

  const title = (
    <span className="awg-status-card-title">
      <ApiOutlined aria-hidden />
      <span>AmneziaWG v2</span>
      {isMobile && (
        <Tag className="awg-version-tag" color={installed ? 'cyan' : 'red'}>
          {version}
        </Tag>
      )}
    </span>
  );

  const extra = (
    <Badge
      status="processing"
      text={running ? 'running' : installed ? 'stopped' : 'unavailable'}
      color={running ? 'green' : installed ? 'orange' : 'red'}
    />
  );

  const actions = [
    <Space
      className="action"
      key="toggle"
      role="button"
      tabIndex={0}
      aria-label={`${toggleLabel} AmneziaWG`}
      onClick={toggleHandler}
      onKeyDown={activateOnKey(toggleHandler)}
    >
      <PoweroffOutlined />
      {!isMobile && <span>{toggleLabel}</span>}
    </Space>,
    <Space
      className="action"
      key="restart"
      role="button"
      tabIndex={0}
      aria-label="Restart AmneziaWG"
      onClick={onRestartAwg}
      onKeyDown={activateOnKey(onRestartAwg)}
    >
      <ReloadOutlined />
      {!isMobile && <span>Restart</span>}
    </Space>,
  ];

  return (
    <>
      <Card hoverable className="awg-status-card" title={title} extra={extra} actions={actions}>
        {!installed && (
          <Alert
            className="awg-status-alert"
            type="warning"
            showIcon
            message="AWG runtime not found"
            description="Install awg and awg-quick on the host to manage AmneziaWG v2 inbounds."
          />
        )}

        <Row gutter={[12, 12]} className="awg-status-metrics">
          <Col xs={12} sm={8}>
            <Statistic
              title="Online peers"
              value={status.awg.onlineCount}
              suffix={<span style={{ fontSize: 14, opacity: 0.65 }}>/ {status.awg.peerCount}</span>}
            />
          </Col>
          <Col xs={12} sm={8}>
            <Statistic title="Interfaces" value={interfaceCount} />
          </Col>
          <Col xs={24} sm={8}>
            <Statistic
              title="Runtime"
              value={!isMobile ? version : (version.length > 14 ? `${version.slice(0, 12)}…` : version)}
              valueStyle={{ fontSize: isMobile ? 16 : 20 }}
            />
          </Col>
        </Row>

        {status.awg.error && (
          <Alert className="awg-status-alert" type="error" showIcon message={status.awg.error} />
        )}

        <div className="awg-status-peers">
          <div className="awg-status-peers-header">
            <span>Runtime peers</span>
            {peers.length > 0 && (
              <Button type="link" size="small" onClick={() => setDrawerOpen(true)}>
                {peers.length > PREVIEW_PEER_LIMIT ? `All ${peers.length}` : 'Details'}
              </Button>
            )}
          </div>

          {previewPeers.length === 0 ? (
            <div className="awg-status-empty">
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={installed ? 'No runtime peers yet' : 'AWG unavailable'}
              />
            </div>
          ) : (
            previewPeers.map((peer, idx) => (
              <PeerRow key={`${peer.publicKey || peer.email || idx}`} peer={peer} index={idx} />
            ))
          )}
        </div>

        {peers.length > 0 && (
          <Row gutter={12} style={{ marginTop: 12 }}>
            <Col span={12}>
              <Statistic
                title="Total upload"
                value={SizeFormatter.sizeFormat(peers.reduce((sum, p) => sum + (p.transferTx || 0), 0))}
                prefix={<ArrowUpOutlined />}
                valueStyle={{ fontSize: 14 }}
              />
            </Col>
            <Col span={12}>
              <Statistic
                title="Total download"
                value={SizeFormatter.sizeFormat(peers.reduce((sum, p) => sum + (p.transferRx || 0), 0))}
                prefix={<ArrowDownOutlined />}
                valueStyle={{ fontSize: 14 }}
              />
            </Col>
          </Row>
        )}
      </Card>

      <Drawer
        title={`AmneziaWG peers (${peers.length})`}
        placement="right"
        width={isMobile ? '100%' : 420}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
      >
        <div className="awg-peer-list">
          {peers.map((peer, idx) => (
            <PeerDrawerRow key={`${peer.publicKey || peer.email || idx}`} peer={peer} index={idx} />
          ))}
        </div>
      </Drawer>
    </>
  );
}
