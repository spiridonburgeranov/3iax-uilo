import { Badge, Card, Space, Tag } from 'antd';
import { ApiOutlined, SettingOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';

import type { Status } from '@/models/status';
import { activateOnKey } from '@/utils/a11y';

interface AwgStatusCardProps {
  status: Status;
  isMobile: boolean;
}

export default function AwgStatusCard({ status, isMobile }: AwgStatusCardProps) {
  const navigate = useNavigate();
  const stateText = status.awg.running ? 'running' : 'stopped';
  const installedText = status.awg.installed ? status.awg.version : 'not installed';

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
        <Space className="action" key="settings" role="button" tabIndex={0} aria-label="AmneziaWG" onClick={() => navigate('/awg')} onKeyDown={activateOnKey(() => navigate('/awg'))}>
          <SettingOutlined />
          {!isMobile && <span>Settings</span>}
        </Space>,
      ]}
    />
  );
}
