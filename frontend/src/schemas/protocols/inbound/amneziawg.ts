import { z } from 'zod';

import { WireguardClientSchema, WireguardInboundPeerSchema } from './wireguard';

const optionalClearedInt = (schema: z.ZodNumber) =>
  z.preprocess((v) => (v == null ? undefined : v), schema.optional());

// AmneziaWG keeps the same peer/client storage shape as WireGuard in the panel.
// Extra obfuscation knobs (Jc/Jmin/...) are optional and applied only by the
// external awg runtime.
export const AmneziawgInboundSettingsSchema = z.object({
  mtu: optionalClearedInt(z.number().int().min(1)),
  secretKey: z.string().min(1),
  dns: z.string().optional(),
  address: z.string().optional(),
  externalInterface: z.string().optional(),
  postUp: z.string().optional(),
  postDown: z.string().optional(),
  peers: z.array(WireguardInboundPeerSchema).default([]),
  clients: z.array(WireguardClientSchema).default([]),
  noKernelTun: z.boolean().default(false),
  domainStrategy: z.enum([
    'ForceIP',
    'ForceIPv4',
    'ForceIPv4v6',
    'ForceIPv6',
    'ForceIPv6v4',
  ]).optional(),
  jc: optionalClearedInt(z.number().int().min(1)),
  jmin: optionalClearedInt(z.number().int().min(1)),
  jmax: optionalClearedInt(z.number().int().min(1)),
  s1: optionalClearedInt(z.number().int().min(0)),
  s2: optionalClearedInt(z.number().int().min(0)),
  s3: optionalClearedInt(z.number().int().min(0)),
  s4: optionalClearedInt(z.number().int().min(0)),
  h1: optionalClearedInt(z.number().int().min(0)),
  h2: optionalClearedInt(z.number().int().min(0)),
  h3: optionalClearedInt(z.number().int().min(0)),
  h4: optionalClearedInt(z.number().int().min(0)),
  i1: z.string().optional(),
  i2: z.string().optional(),
  i3: z.string().optional(),
  i4: z.string().optional(),
  i5: z.string().optional(),
});
export type AmneziawgInboundSettings = z.infer<typeof AmneziawgInboundSettingsSchema>;
