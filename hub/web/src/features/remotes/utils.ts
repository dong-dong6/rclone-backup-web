import type { RemotePresetKey } from './constants';

export interface ParsedRcloneConfig {
  sectionNames: string[];
  options: Record<string, string>;
}

export const parseRcloneConfig = (configData: string): ParsedRcloneConfig => {
  const sectionNames: string[] = [];
  const options: Record<string, string> = {};

  let currentSectionIndex = -1;
  const lines = configData.split(/\r?\n/);

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line) continue;
    if (line.startsWith('#') || line.startsWith(';')) continue;

    const sectionMatch = line.match(/^\[([^\]]+)\]$/);
    if (sectionMatch) {
      sectionNames.push(sectionMatch[1]);
      currentSectionIndex = sectionNames.length - 1;
      continue;
    }

    const eqIndex = line.indexOf('=');
    if (eqIndex === -1) continue;

    const key = line.slice(0, eqIndex).trim();
    const value = line.slice(eqIndex + 1).trim();
    if (!key) continue;

    // Only parse the first section (or section-less config) for safety.
    if (currentSectionIndex <= 0) {
      options[key] = value;
    }
  }

  return { sectionNames, options };
};

export const normalizeTokenJson = (value: string): string => {
  const trimmed = value.trim();
  if (!trimmed) return '';
  try {
    return JSON.stringify(JSON.parse(trimmed));
  } catch {
    return trimmed;
  }
};

export const isValidRemoteName = (name: string): boolean => {
  const trimmed = name.trim();
  if (!trimmed) return false;
  if (/[\r\n\[\]:]/.test(trimmed)) return false;
  if (trimmed.length > 128) return false;
  return true;
};

export const buildRcloneConfigSection = (
  remoteName: string,
  type: string,
  options: Record<string, string>
): string => {
  const lines: string[] = [`[${remoteName}]`, `type = ${type}`];

  for (const [key, value] of Object.entries(options)) {
    const trimmedValue = (value ?? '').toString().trim();
    if (!trimmedValue) continue;
    if (key === 'type') continue;
    lines.push(`${key} = ${trimmedValue}`);
  }

  return lines.join('\n') + '\n';
};

export const normalizeRawConfigData = (
  remoteName: string,
  rawConfig: string
): { ok: true; type: string; configData: string } | { ok: false; error: 'multiple_sections' | 'missing_type' } => {
  const parsed = parseRcloneConfig(rawConfig);
  if (parsed.sectionNames.length > 1) {
    return { ok: false, error: 'multiple_sections' };
  }

  const type = parsed.options.type?.trim();
  if (!type) {
    return { ok: false, error: 'missing_type' };
  }

  const options = { ...parsed.options };
  delete options.type;

  return {
    ok: true,
    type,
    configData: buildRcloneConfigSection(remoteName, type, options),
  };
};

export const detectPreset = (
  type: string | undefined,
  options: Record<string, string>
): RemotePresetKey | null => {
  if (!type) return null;

  if (type === 'drive') return 'drive';
  if (type === 'onedrive') return 'onedrive';

  if (type === 's3') {
    const provider = options.provider?.trim();
    if (provider === 'Cloudflare') return 's3_cloudflare_r2';
    if (provider === 'Alibaba') return 's3_alibaba_oss';
    if (provider === 'TencentCOS') return 's3_tencent_cos';
  }

  return null;
};
