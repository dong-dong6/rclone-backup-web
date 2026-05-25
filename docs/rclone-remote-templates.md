# rclone Remote Templates (for `config_data`)

This project stores an encrypted `config_data` string per remote, and the Agent writes it as a temporary `rclone.conf` file when running `rclone` commands.

Important:
- `config_data` must contain **exactly one** section header like `[remote-name]`.
- The section name must match the Remote `name` you create in the UI (because the Agent runs `rclone ... <name>:`).
- Remote `name` must be a single line and must not include `[` `]` `:` (max length 128).

## One-click OAuth (Drive / OneDrive)

The Web UI supports “One-click OAuth” for Google Drive and OneDrive. To use it, you need to create your own OAuth app and add these redirect URIs:

- Drive callback: `https://<your-host>/api/v1/oauth/drive/callback`
- OneDrive callback: `https://<your-host>/api/v1/oauth/onedrive/callback`

If you are behind a reverse proxy, make sure it forwards `X-Forwarded-Proto` and `X-Forwarded-Host` so the Hub can build correct callback URLs.

Note: OAuth flows are stored in Hub process memory. If you run multiple Hub replicas, you must use sticky sessions (or change the implementation to use shared storage) to avoid “flow not found” on callback.

## Google Drive (`type = drive`)

Get OAuth token JSON either via the UI “One-click OAuth”, or on your own machine:
- `rclone authorize drive`

Example:
```ini
[my-gdrive]
type = drive
scope = drive
token = {"access_token":"...","token_type":"Bearer","refresh_token":"...","expiry":"..."}

# optional:
# client_id = ...
# client_secret = ...
# team_drive = ...
# root_folder_id = ...
```

## OneDrive (`type = onedrive`)

Get OAuth token JSON either via the UI “One-click OAuth”, or on your own machine:
- `rclone authorize onedrive`

Example:
```ini
[my-onedrive]
type = onedrive
region = global
token = {"access_token":"...","token_type":"Bearer","refresh_token":"...","expiry":"..."}

# optional:
# client_id = ...
# client_secret = ...
# drive_type = personal | business | documentLibrary
# drive_id = ...
# root_folder_id = ...
```

## Cloudflare R2 (`type = s3`, `provider = Cloudflare`)

Example:
```ini
[my-r2]
type = s3
provider = Cloudflare
access_key_id = ...
secret_access_key = ...
endpoint = https://<ACCOUNT_ID>.r2.cloudflarestorage.com
region = auto
```

## Alibaba Cloud OSS (`type = s3`, `provider = Alibaba`)

Example:
```ini
[my-aliyun-oss]
type = s3
provider = Alibaba
access_key_id = ...
secret_access_key = ...
endpoint = oss-cn-hangzhou.aliyuncs.com
```

## Tencent Cloud COS (`type = s3`, `provider = TencentCOS`)

Example:
```ini
[my-tencent-cos]
type = s3
provider = TencentCOS
access_key_id = ...
secret_access_key = ...
endpoint = cos.ap-beijing.myqcloud.com
```
