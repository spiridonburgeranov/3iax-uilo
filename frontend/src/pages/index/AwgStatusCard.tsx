import { Badge, Card, Popover, Space, Tag } from 'antd';
import { ApiOutlined, PoweroffOutlined, ReloadOutlined, SettingOutlined, TeamOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

import type { Status } from '@/models/status';
import { SizeFormatter } from '@/utils';
import { activateOnKey } from '@/utils/a11y';

interface AwgStatusCardProps {
  status: Status;
  isMobile: boolean;
  onStartAwg: () => void;
  onStopAwg: () => void;
  onRestartAwg: () => void;
}

export default function AwgStatusCard({
  status,
  isMobile,
  onStartAwg,
  onStopAwg,
  onRestartAwg,
}: AwgStatusCardProps) {
  const navigate = useNavigate();
  const stateText = status.awg.running ? 'running' : 'stopped';
  const installedText = status.awg.installed ? status.awg.version : 'not installed';
  const toggleHandler = status.awg.running ? onStopAwg : onStartAwg;
  const toggleLabel = status.awg.running ? 'Stop' : 'Start';
  const peerText = `${status.awg.onlineCount}/${status.awg.peerCount}`;
  const peers = status.awg.peers || [];
  const peerContent = (
    <div className="awg-peer-list">
      {status.awg.error && <Tag color="red">{status.awg.error}</Tag>}
      {peers.length === 0 && !status.awg.error ? (
        <Tag>no runtime peers</Tag>
      ) : peers.map((peer, idx) => (
        <div className="awg-peer-row" key={`${peer.publicKey || idx}`}>
          <div className="awg-peer-main">
            <span>{peer.email || peer.publicKey || `peer-${idx + 1}`}</span>
            <Tag color={peer.online ? 'green' : 'default'}>{peer.online ? 'online' : 'idle'}</Tag>
          </div>
          <div className="awg-peer-meta">{peer.endpoint || 'no endpoint'}</div>
          <div className="awg-peer-meta">{(peer.allowedIPs || []).join(', ') || 'no allowed IPs'}</div>
          <div className="awg-peer-meta">
            ↑ {SizeFormatter.sizeFormat(peer.transferTx || 0)} / ↓ {SizeFormatter.sizeFormat(peer.transferRx || 0)}
          </div>
          {peer.latestHandshake ? (
            <div className="awg-peer-meta">handshake {new Date(peer.latestHandshake * 1000).toLocaleString()}</div>
          ) : (
            <div className="awg-peer-meta">no handshake</div>
          )}
        </div>
      ))}
    </div>
  );

  return (
    <Card
      hoverable
      title={
        <Space>
          <span>AmneziaWG</span>
          {isMobile && <Tag color={status.awg.installed ? 'green' : 'red'}>{installedText}</Tag>}
        </Space>
      }
      extra={<Badge status="processing" text={stateText} color={status.awg.running ? 'green' : 'orange'} />}
      actions={[
        <Space className="action" key="version">
          <ApiOutlined />
          {!isMobile && <span>{installedText}</span>}
        </Space>,
        <Popover key="peers" title="Runtime peers" content={peerContent} trigger="click">
          <Space className="action" role="button" tabIndex={0} aria-label="AmneziaWG peers">
            <TeamOutlined />
            {!isMobile && <span>{peerText} peers</span>}
          </Space>
        </Popover>,
        <Space className="action" key="toggle" role="button" tabIndex={0} aria-label={`${toggleLabel} AmneziaWG`} onClick={toggleHandler} onKeyDown={activateOnKey(toggleHandler)}>
          <PoweroffOutlined />
          {!isMobile && <span>{toggleLabel}</span>}
        </Space>,
        <Space className="action" key="restart" role="button" tabIndex={0} aria-label="Restart AmneziaWG" onClick={onRestartAwg} onKeyDown={activateOnKey(onRestartAwg)}>
          <ReloadOutlined />
          {!isMobile && <span>Restart</span>}
        </Space>,
        <Space className="action" key="settings" role="button" tabIndex={0} aria-label="AmneziaWG" onClick={() => navigate('/awg')} onKeyDown={activateOnKey(() => navigate('/awg'))}>
          <SettingOutlined />
          {!isMobile && <span>Settings</span>}
        </Space>,
      ]}
    />
  );
}
