import { Wireguard } from '@/utils';
import { generateAwgObfuscationParams } from '@/lib/awg/obfuscation';

const DEFAULT_DNS = '1.1.1.1,2606:4700:4700::1111';
const DEFAULT_ADDRESS = '10.66.66.1/24';

export interface LocalAwgProvisionResult {
  remark: string;
  port: number;
  enable: boolean;
  tag: string;
  publicKey: string;
  interfaceName: string;
  configPath: string;
  settings: Record<string, unknown>;
}

export function buildLocalAwgProvision(interfaceIndex = 0): LocalAwgProvisionResult {
  const keypair = Wireguard.generateKeypair();
  const obf = generateAwgObfuscationParams();
  const iface = `awg${interfaceIndex}`;
  const port = 10000 + Math.floor(Math.random() * 55535);
  return {
    remark: `AmneziaWG ${iface}`,
    port,
    enable: true,
    tag: `inbound-${iface}`,
    publicKey: keypair.publicKey,
    interfaceName: iface,
    configPath: `/etc/amnezia/amneziawg/${iface}.conf`,
    settings: {
      secretKey: keypair.privateKey,
      address: DEFAULT_ADDRESS,
      dns: DEFAULT_DNS,
      awgInterface: iface,
      mtu: 1420,
      jc: obf.jc,
      jmin: obf.jmin,
      jmax: obf.jmax,
      s1: obf.s1,
      s2: obf.s2,
      s3: obf.s3,
      s4: obf.s4,
      h1: obf.h1,
      h2: obf.h2,
      h3: obf.h3,
      h4: obf.h4,
      i1: obf.i1,
      i2: obf.i2,
      i3: obf.i3,
      i4: obf.i4,
      i5: obf.i5,
      postUp: '',
      postDown: '',
      clients: [],
      peers: [],
    },
  };
}
