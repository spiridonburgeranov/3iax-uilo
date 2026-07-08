import { Tag } from 'antd';

import { SizeFormatter } from '@/utils';
import type { ClientSpeedEntry } from '@/hooks/useClients';
import { CLIENT_SPEED_STALE_MS } from '@/lib/traffic/poll-interval';

export type { ClientSpeedEntry };

export function isActiveSpeed(speed?: ClientSpeedEntry): speed is ClientSpeedEntry {
  if (!speed || (speed.up <= 0 && speed.down <= 0)) return false;
  if (speed.at != null && Date.now() - speed.at > CLIENT_SPEED_STALE_MS) return false;
  return true;
}

interface ClientSpeedTagProps {
  speed: ClientSpeedEntry;
}

export function ClientSpeedTag({ speed }: ClientSpeedTagProps) {
  return (
    <Tag color="blue">
      ↑ {SizeFormatter.speedFormat(speed.up)}
      {' / '}
      ↓ {SizeFormatter.speedFormat(speed.down)}
    </Tag>
  );
}

export default ClientSpeedTag;
