// Remote related types

export interface RcloneRemote {
  id: string;
  name: string;
  type?: string;
  last_test_at?: string;
  last_test_success?: boolean;
  last_test_message?: string;
  last_test_error?: string;
  last_test_duration_ms?: number;
  created_at: string;
  updated_at: string;
}

export interface RcloneRemoteDetail extends RcloneRemote {
  config_data: string;
}

export type RemotePresetKey =
  | 'drive'
  | 'onedrive'
  | 's3_cloudflare_r2'
  | 's3_alibaba_oss'
  | 's3_tencent_cos';

export interface RemoteTestResponse {
  success?: boolean;
  message?: string;
  error?: string;
  output?: string;
  duration_ms?: number;
}

export interface OAuthConfig {
  name: string;
  [key: string]: string;
}

export interface PresetField {
  key: string;
  label: string;
  type: 'text' | 'password' | 'textarea' | 'select';
  required?: boolean;
  placeholder?: string;
  options?: Array<{ label: string; value: string }>;
}

export interface RemotePreset {
  type: string;
  label: string;
  fields: PresetField[];
}
