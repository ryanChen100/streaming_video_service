#!/bin/bash

# 定義 .env 文件的路徑
ENV_FILE="../.env"

# 檢查 .env 文件是否存在
if [[ ! -f $ENV_FILE ]]; then
  echo ".env file not found at $ENV_FILE"
  exit 1
fi

# 讀取 .env 文件的內容
REDIS_SENTINEL_PORTS=()
REDIS_MASTER_IP=""
REDIS_MASTER_NAME="mymaster" # 預設值為 mymaster

while IFS= read -r line || [[ -n $line ]]; do
  # 去除行首和行尾空白
  line=$(echo "$line" | xargs)

  # 跳過空行和註解
  [[ -z $line || $line == \#* ]] && continue

  # 解析 REDIS_SENTINEL* 變量
  if [[ $line =~ ^REDIS_SENTINEL([0-9]+)_PORT=([0-9]+)$ ]]; then
    SENTINEL_INDEX=${BASH_REMATCH[1]}
    SENTINEL_PORT=${BASH_REMATCH[2]}
    REDIS_SENTINEL_PORTS+=("$SENTINEL_INDEX:$SENTINEL_PORT")
  fi

  # 解析 REDIS_MASTER_IP
  if [[ $line =~ ^REDIS_MASTER_IP=([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
    REDIS_MASTER_IP=${BASH_REMATCH[1]}
  fi

  # 解析 REDIS_MASTER_NAME
  if [[ $line =~ ^REDIS_MASTER_NAME=(.+)$ ]]; then
    REDIS_MASTER_NAME=${BASH_REMATCH[1]}
  fi

done < "$ENV_FILE"

# 確保讀取到 REDIS_MASTER_IP
if [[ -z $REDIS_MASTER_IP ]]; then
  echo "REDIS_MASTER_IP not found in .env file"
  exit 1
fi

# 生成 Sentinel 配置文件
for entry in "${REDIS_SENTINEL_PORTS[@]}"; do
  SENTINEL_INDEX=${entry%%:*}
  SENTINEL_PORT=${entry##*:}

  CONFIG_FILE="./redis_sentinel/sentinel-${SENTINEL_INDEX}.conf"
  LOG_FILE="sentinel-${SENTINEL_INDEX}.log"

  cat > "$CONFIG_FILE" <<EOF
port $SENTINEL_PORT
# Sentinel 的工作目錄
dir "/data"
logfile "$LOG_FILE"

# 配置監控的主節點
sentinel monitor $REDIS_MASTER_NAME $REDIS_MASTER_IP 6379 2
# 如果 Redis 主節點有密碼，取消註釋以下行並添加密碼
# sentinel auth-pass $REDIS_MASTER_NAME your_master_password

# 故障切換設置
sentinel down-after-milliseconds $REDIS_MASTER_NAME 5000
sentinel failover-timeout $REDIS_MASTER_NAME 10000
sentinel parallel-syncs $REDIS_MASTER_NAME 1
EOF

  echo "Generated: $CONFIG_FILE"
done
