#!/bin/bash
# 需要备份的文件夹路径
SOURCE_DIR="{{ .SourceDir }}"
# rclone配置名称和目标路径
RCLONE_REMOTE={{ .RcloneRemoteStr }}
# 保留的最大备份数量
MAX_BACKUPS={{ .MaxBackups }}
# 是否分卷
IS_SPLIST="{{ .IsSplit }}"
# 是否加密
IS_Encrypt="{{ .IsEncrypt }}"
# 加密密钥
EncryptionPassword="{{ .EncryptionPassword }}"
# 获取脚本所在目录路径
SCRIPT_DIR=$(dirname "$(readlink -f "$0")")
# 备份文件临时存放路径
TEMP_DIR="$SCRIPT_DIR/temp"
# 日志文件路径
LOG_FILE="$SCRIPT_DIR/backup.log"
# 当前时间，用于备份文件命名
DATE=$(date +"%Y-%m-%d_%H-%M-%S")

# 创建临时目录（如果不存在）
mkdir -p "$TEMP_DIR"

echo "[$(date +"%Y-%m-%d_%H-%M-%S")] 开始压缩 $SOURCE_DIR" >> "$LOG_FILE"

ARCHIVE_NAME="${TEMP_DIR}/$(basename "$SOURCE_DIR")-${DATE}.tar.gz"
echo "[$(date +"%Y-%m-%d_%H-%M-%S")] 压缩开始" >> "$LOG_FILE"
tar -czf "$ARCHIVE_NAME" "$SOURCE_DIR" >> /dev/null 2>&1


# 检查文件是否成功创建
if [ -f "$ARCHIVE_NAME" ]; then
    echo "[$(date +"%Y-%m-%d_%H-%M-%S")] tar 压缩成功: $ARCHIVE_NAME" >> "$LOG_FILE"
else
    echo "[$(date +"%Y-%m-%d_%H-%M-%S")] tar 压缩失败: $ARCHIVE_NAME 未创建" >> "$LOG_FILE"
    exit 1
fi
# 检查是否需要加密
if [ "$IS_Encrypt" == "true" ]; then
    echo "[$(date +"%Y-%m-%d_%H-%M-%S")] 开始加密 $ARCHIVE_NAME" >> "$LOG_FILE"
    # 定义加密后的文件名
    ENCRYPTED_ARCHIVE_NAME="${ARCHIVE_NAME}.enc"
    # 使用 openssl 进行加密
    openssl enc -aes-256-cbc -salt -in "$ARCHIVE_NAME" -out "$ENCRYPTED_ARCHIVE_NAME" -k "$EncryptionPassword" >> "$LOG_FILE" 2>&1

    if [ $? -eq 0 ]; then
        echo "[$(date +"%Y-%m-%d_%H-%M-%S")] 加密完成: $ENCRYPTED_ARCHIVE_NAME" >> "$LOG_FILE"
        rm "$ARCHIVE_NAME"  # 删除未加密的 tar 文件
        ARCHIVE_NAME="$ENCRYPTED_ARCHIVE_NAME"  # 更新变量指向加密后的文件
    else
        echo "[$(date +"%Y-%m-%d_%H-%M-%S")] 加密失败" >> "$LOG_FILE"
        exit 1
    fi
fi

# 遍历每个远程位置
for REMOTE in "${RCLONE_REMOTE[@]}"; do
    # 删除云端最早的备份（超过最大保留数量）
    EXISTING_BACKUPS=$(rclone lsf "$REMOTE" | grep "$(basename "$SOURCE_DIR")-.*\.tar\.gz" | sort)
    NUM_BACKUPS=$(echo "$EXISTING_BACKUPS" | wc -l)

    if [ "$NUM_BACKUPS" -gt "$MAX_BACKUPS" ]; then
        NUM_TO_DELETE=$((NUM_BACKUPS - MAX_BACKUPS))
        BACKUPS_TO_DELETE=$(echo "$EXISTING_BACKUPS" | head -n "$NUM_TO_DELETE")

        for BACKUP in $BACKUPS_TO_DELETE; do
            echo "[$(date +"%Y-%m-%d_%H-%M-%S")] Deleting old backup: $BACKUP on $REMOTE" >> "$LOG_FILE"
            rclone delete "$REMOTE/$BACKUP" >> "$LOG_FILE" 2>&1
        done
    fi

    # 上传新的备份到当前远程位置
    echo "[$(date +"%Y-%m-%d_%H-%M-%S")] 开始上传至 $ARCHIVE_NAME to $REMOTE" >> "$LOG_FILE"
    rclone copy "$ARCHIVE_NAME" "$REMOTE/" --s3-no-check-bucket --log-file="$LOG_FILE" --log-level INFO
    UPLOAD_STATUS=$?

    # 检查上传是否成功
    if [ $UPLOAD_STATUS -eq 0 ]; then
        echo "[$(date +"%Y-%m-%d_%H-%M-%S")] 上传 $ARCHIVE_NAME to $REMOTE 成功" >> "$LOG_FILE"
    else
        echo "[$(date +"%Y-%m-%d_%H-%M-%S")] Failed to upload $ARCHIVE_NAME to $REMOTE" >> "$LOG_FILE"
    fi
done


# 记录备份过程完成状态
if [ $UPLOAD_STATUS -eq 0 ]; then
    echo "[$(date +"%Y-%m-%d_%H-%M-%S")] 备份完成" >> "$LOG_FILE"
else
    echo "[$(date +"%Y-%m-%d_%H-%M-%S")] 备份出现异常" >> "$LOG_FILE"
fi
