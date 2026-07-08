const MAX_AWG_HEADER_VALUE = 2_147_483_647;

export interface AwgObfuscationParams {
  jc: number;
  jmin: number;
  jmax: number;
  s1: number;
  s2: number;
  s3: number;
  s4: number;
  h1: string;
  h2: string;
  h3: string;
  h4: string;
  i1: string;
  i2: string;
  i3: string;
  i4: string;
  i5: string;
}

function randomInt(maxExclusive: number): number {
  if (maxExclusive <= 0) return 0;
  const buffer = new Uint32Array(1);
  crypto.getRandomValues(buffer);
  return buffer[0] % maxExclusive;
}

function uniqueInts(values: number[]): boolean {
  return new Set(values).size === values.length;
}

function shuffleStrings(values: string[]): void {
  for (let i = values.length - 1; i > 0; i -= 1) {
    const j = randomInt(i + 1);
    [values[i], values[j]] = [values[j], values[i]];
  }
}

function generateHeaderRanges(): [string, string, string, string] {
  let current = 150_000_000 + randomInt(50_000_000);
  const ranges: string[] = [];
  for (let i = 0; i < 4; i += 1) {
    const start = current;
    let end = start + 50_000_000 + randomInt(100_000_000);
    if (end > MAX_AWG_HEADER_VALUE) end = MAX_AWG_HEADER_VALUE;
    ranges.push(`${start}-${end}`);
    current = end + 10_000_000 + randomInt(20_000_000);
    if (current > MAX_AWG_HEADER_VALUE - 100_000_000) {
      current = 150_000_000 + randomInt(50_000_000);
    }
  }
  shuffleStrings(ranges);
  return [ranges[0], ranges[1], ranges[2], ranges[3]];
}

export function generateAwgObfuscationParams(): AwgObfuscationParams {
  const jc = 3 + randomInt(4);
  const jmin = 64 + randomInt(50);
  let jmax = jmin + 50 + randomInt(100);
  if (jmax > 1024) jmax = 1024;
  if (jmax < jmin) jmax = jmin + 1;

  let s1 = 15;
  let s2 = 25;
  let s3 = 35;
  let s4 = 15;
  for (let attempt = 0; attempt < 200; attempt += 1) {
    s1 = 15 + randomInt(49);
    s2 = 15 + randomInt(49);
    s3 = 10 + randomInt(54);
    s4 = 1 + randomInt(15);
    if (!uniqueInts([s1, s2, s3, s4])) continue;
    if (s1 + 148 === s2 + 92 || s3 + 64 === s1 + 148 || s3 + 64 === s2 + 92) continue;
    break;
  }

  const [h1, h2, h3, h4] = generateHeaderRanges();
  return {
    jc,
    jmin,
    jmax,
    s1,
    s2,
    s3,
    s4,
    h1,
    h2,
    h3,
    h4,
    i1: `<r ${15 + randomInt(26)}>`,
    i2: `<r ${10 + randomInt(20)}>`,
    i3: `<r ${10 + randomInt(20)}>`,
    i4: `<r ${10 + randomInt(20)}>`,
    i5: `<r ${10 + randomInt(20)}>`,
  };
}
