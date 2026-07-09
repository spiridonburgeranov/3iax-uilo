import {
  Alert,
  Badge,
  Card,
  Col,
  Row,
  Space,
  Statistic,
  Tag,
} from 'antd';
import {
  ApiOutlined,
  PoweroffOutlined,
  ReloadOutlined,
} from '@ant-design/icons';

import type { Status } from '@/models/status';
import { activateOnKey } from '@/utils/a11y';
import './AwgStatusCard.css';

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
  const running = status.awg.running;
  const installed = status.awg.installed;
  const version = installed ? status.awg.version : 'not installed';
  const toggleHandler = running ? onStopAwg : onStartAwg;
  const toggleLabel = running ? 'Stop' : 'Start';

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
          <Statistic title="Configured peers" value={status.awg.peerCount} />
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
    </Card>
  );
}
