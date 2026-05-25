export type RemotePresetKey = 'drive' | 'onedrive' | 's3_cloudflare_r2' | 's3_alibaba_oss' | 's3_tencent_cos';

export type FieldKind = 'text' | 'password' | 'textarea' | 'select';

export interface FieldDef {
  key: string;
  label: string;
  kind: FieldKind;
  placeholder?: string;
  help?: string;
  required?: boolean;
  options?: Array<{ value: string; label: string }>;
}

export interface RemotePreset {
  key: RemotePresetKey;
  label: string;
  type: 'drive' | 'onedrive' | 's3';
  constantOptions: Record<string, string>;
  fields: FieldDef[];
  initialValues: Record<string, string>;
}

export const alibabaEndpoints = [
  { value: 'oss-accelerate.aliyuncs.com', label: 'oss-accelerate.aliyuncs.com' },
  { value: 'oss-accelerate-overseas.aliyuncs.com', label: 'oss-accelerate-overseas.aliyuncs.com' },
  { value: 'oss-cn-hangzhou.aliyuncs.com', label: 'oss-cn-hangzhou.aliyuncs.com' },
  { value: 'oss-cn-shanghai.aliyuncs.com', label: 'oss-cn-shanghai.aliyuncs.com' },
  { value: 'oss-cn-qingdao.aliyuncs.com', label: 'oss-cn-qingdao.aliyuncs.com' },
  { value: 'oss-cn-beijing.aliyuncs.com', label: 'oss-cn-beijing.aliyuncs.com' },
  { value: 'oss-cn-zhangjiakou.aliyuncs.com', label: 'oss-cn-zhangjiakou.aliyuncs.com' },
  { value: 'oss-cn-huhehaote.aliyuncs.com', label: 'oss-cn-huhehaote.aliyuncs.com' },
  { value: 'oss-cn-wulanchabu.aliyuncs.com', label: 'oss-cn-wulanchabu.aliyuncs.com' },
  { value: 'oss-cn-shenzhen.aliyuncs.com', label: 'oss-cn-shenzhen.aliyuncs.com' },
  { value: 'oss-cn-heyuan.aliyuncs.com', label: 'oss-cn-heyuan.aliyuncs.com' },
  { value: 'oss-cn-guangzhou.aliyuncs.com', label: 'oss-cn-guangzhou.aliyuncs.com' },
  { value: 'oss-cn-chengdu.aliyuncs.com', label: 'oss-cn-chengdu.aliyuncs.com' },
  { value: 'oss-cn-hongkong.aliyuncs.com', label: 'oss-cn-hongkong.aliyuncs.com' },
  { value: 'oss-us-west-1.aliyuncs.com', label: 'oss-us-west-1.aliyuncs.com' },
  { value: 'oss-us-east-1.aliyuncs.com', label: 'oss-us-east-1.aliyuncs.com' },
  { value: 'oss-ap-southeast-1.aliyuncs.com', label: 'oss-ap-southeast-1.aliyuncs.com' },
  { value: 'oss-ap-southeast-2.aliyuncs.com', label: 'oss-ap-southeast-2.aliyuncs.com' },
  { value: 'oss-ap-southeast-3.aliyuncs.com', label: 'oss-ap-southeast-3.aliyuncs.com' },
  { value: 'oss-ap-southeast-5.aliyuncs.com', label: 'oss-ap-southeast-5.aliyuncs.com' },
  { value: 'oss-ap-northeast-1.aliyuncs.com', label: 'oss-ap-northeast-1.aliyuncs.com' },
  { value: 'oss-ap-south-1.aliyuncs.com', label: 'oss-ap-south-1.aliyuncs.com' },
  { value: 'oss-eu-central-1.aliyuncs.com', label: 'oss-eu-central-1.aliyuncs.com' },
  { value: 'oss-eu-west-1.aliyuncs.com', label: 'oss-eu-west-1.aliyuncs.com' },
  { value: 'oss-me-east-1.aliyuncs.com', label: 'oss-me-east-1.aliyuncs.com' },
];

export const tencentEndpoints = [
  { value: 'cos.ap-beijing.myqcloud.com', label: 'cos.ap-beijing.myqcloud.com' },
  { value: 'cos.ap-nanjing.myqcloud.com', label: 'cos.ap-nanjing.myqcloud.com' },
  { value: 'cos.ap-shanghai.myqcloud.com', label: 'cos.ap-shanghai.myqcloud.com' },
  { value: 'cos.ap-guangzhou.myqcloud.com', label: 'cos.ap-guangzhou.myqcloud.com' },
  { value: 'cos.ap-chengdu.myqcloud.com', label: 'cos.ap-chengdu.myqcloud.com' },
  { value: 'cos.ap-chongqing.myqcloud.com', label: 'cos.ap-chongqing.myqcloud.com' },
  { value: 'cos.ap-hongkong.myqcloud.com', label: 'cos.ap-hongkong.myqcloud.com' },
  { value: 'cos.ap-singapore.myqcloud.com', label: 'cos.ap-singapore.myqcloud.com' },
  { value: 'cos.ap-mumbai.myqcloud.com', label: 'cos.ap-mumbai.myqcloud.com' },
  { value: 'cos.ap-seoul.myqcloud.com', label: 'cos.ap-seoul.myqcloud.com' },
  { value: 'cos.ap-bangkok.myqcloud.com', label: 'cos.ap-bangkok.myqcloud.com' },
  { value: 'cos.ap-tokyo.myqcloud.com', label: 'cos.ap-tokyo.myqcloud.com' },
  { value: 'cos.na-siliconvalley.myqcloud.com', label: 'cos.na-siliconvalley.myqcloud.com' },
  { value: 'cos.na-ashburn.myqcloud.com', label: 'cos.na-ashburn.myqcloud.com' },
  { value: 'cos.na-toronto.myqcloud.com', label: 'cos.na-toronto.myqcloud.com' },
  { value: 'cos.eu-frankfurt.myqcloud.com', label: 'cos.eu-frankfurt.myqcloud.com' },
  { value: 'cos.eu-moscow.myqcloud.com', label: 'cos.eu-moscow.myqcloud.com' },
  { value: 'cos.accelerate.myqcloud.com', label: 'cos.accelerate.myqcloud.com' },
];

export const createPresets = (t: (key: string) => string): Record<RemotePresetKey, RemotePreset> => ({
  drive: {
    key: 'drive',
    label: t('remotes.preset.drive'),
    type: 'drive',
    constantOptions: {},
    initialValues: {
      client_id: '',
      client_secret: '',
      scope: '',
      token: '',
      team_drive: '',
      root_folder_id: '',
    },
    fields: [
      { key: 'client_id', label: t('remotes.fields.client_id'), kind: 'text', placeholder: t('remotes.placeholders.client_id') },
      { key: 'client_secret', label: t('remotes.fields.client_secret'), kind: 'password', placeholder: t('remotes.placeholders.client_secret') },
      {
        key: 'scope',
        label: t('remotes.fields.scope'),
        kind: 'select',
        options: [
          { value: '', label: t('remotes.values.default') },
          { value: 'drive', label: 'drive' },
          { value: 'drive.readonly', label: 'drive.readonly' },
          { value: 'drive.file', label: 'drive.file' },
          { value: 'drive.appfolder', label: 'drive.appfolder' },
          { value: 'drive.metadata.readonly', label: 'drive.metadata.readonly' },
        ],
      },
      {
        key: 'token',
        label: t('remotes.fields.token'),
        kind: 'textarea',
        required: true,
        help: t('remotes.hints.token_drive'),
        placeholder: '{"access_token":"...","token_type":"Bearer","refresh_token":"...","expiry":"..."}',
      },
      { key: 'team_drive', label: t('remotes.fields.team_drive'), kind: 'text', placeholder: t('remotes.placeholders.team_drive') },
      { key: 'root_folder_id', label: t('remotes.fields.root_folder_id'), kind: 'text', placeholder: t('remotes.placeholders.root_folder_id') },
    ],
  },
  onedrive: {
    key: 'onedrive',
    label: t('remotes.preset.onedrive'),
    type: 'onedrive',
    constantOptions: {},
    initialValues: {
      client_id: '',
      client_secret: '',
      region: 'global',
      token: '',
      drive_type: '',
      drive_id: '',
      root_folder_id: '',
    },
    fields: [
      { key: 'client_id', label: t('remotes.fields.client_id'), kind: 'text', placeholder: t('remotes.placeholders.client_id') },
      { key: 'client_secret', label: t('remotes.fields.client_secret'), kind: 'password', placeholder: t('remotes.placeholders.client_secret') },
      {
        key: 'region',
        label: t('remotes.fields.region'),
        kind: 'select',
        options: [
          { value: 'global', label: 'global' },
          { value: 'us', label: 'us' },
          { value: 'de', label: 'de' },
          { value: 'cn', label: 'cn' },
        ],
      },
      {
        key: 'token',
        label: t('remotes.fields.token'),
        kind: 'textarea',
        required: true,
        help: t('remotes.hints.token_onedrive'),
        placeholder: '{"access_token":"...","token_type":"Bearer","refresh_token":"...","expiry":"..."}',
      },
      { key: 'drive_type', label: t('remotes.fields.drive_type'), kind: 'text', placeholder: 'personal | business | documentLibrary' },
      { key: 'drive_id', label: t('remotes.fields.drive_id'), kind: 'text' },
      { key: 'root_folder_id', label: t('remotes.fields.root_folder_id'), kind: 'text', placeholder: t('remotes.placeholders.root_folder_id') },
    ],
  },
  s3_cloudflare_r2: {
    key: 's3_cloudflare_r2',
    label: t('remotes.preset.cloudflare_r2'),
    type: 's3',
    constantOptions: { provider: 'Cloudflare', region: 'auto', no_check_bucket: 'true' },
    initialValues: { access_key_id: '', secret_access_key: '', endpoint: '' },
    fields: [
      { key: 'access_key_id', label: t('remotes.fields.access_key_id'), kind: 'text', required: true },
      { key: 'secret_access_key', label: t('remotes.fields.secret_access_key'), kind: 'password', required: true },
      { key: 'endpoint', label: t('remotes.fields.endpoint'), kind: 'text', required: true, placeholder: 'https://<ACCOUNT_ID>.r2.cloudflarestorage.com' },
    ],
  },
  s3_alibaba_oss: {
    key: 's3_alibaba_oss',
    label: t('remotes.preset.aliyun_oss'),
    type: 's3',
    constantOptions: { provider: 'Alibaba' },
    initialValues: { access_key_id: '', secret_access_key: '', endpoint: '' },
    fields: [
      { key: 'access_key_id', label: t('remotes.fields.access_key_id'), kind: 'text', required: true },
      { key: 'secret_access_key', label: t('remotes.fields.secret_access_key'), kind: 'password', required: true },
      { key: 'endpoint', label: t('remotes.fields.endpoint'), kind: 'select', required: true, options: [{ value: '', label: t('remotes.values.custom') }, ...alibabaEndpoints] },
    ],
  },
  s3_tencent_cos: {
    key: 's3_tencent_cos',
    label: t('remotes.preset.tencent_cos'),
    type: 's3',
    constantOptions: { provider: 'TencentCOS' },
    initialValues: { access_key_id: '', secret_access_key: '', endpoint: '' },
    fields: [
      { key: 'access_key_id', label: t('remotes.fields.access_key_id'), kind: 'text', required: true },
      { key: 'secret_access_key', label: t('remotes.fields.secret_access_key'), kind: 'password', required: true },
      { key: 'endpoint', label: t('remotes.fields.endpoint'), kind: 'select', required: true, options: [{ value: '', label: t('remotes.values.custom') }, ...tencentEndpoints] },
    ],
  },
});
