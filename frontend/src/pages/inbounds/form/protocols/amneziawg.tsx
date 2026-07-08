import WireguardFields from './wireguard';

interface AmneziawgFieldsProps {
  wgPubKey: string;
  regenInboundWg: () => void;
}

export default function AmneziawgFields(props: AmneziawgFieldsProps) {
  return <WireguardFields {...props} />;
}
