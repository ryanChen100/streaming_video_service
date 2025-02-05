#!/usr/bin/env bash

# 停止於任何錯誤
set -e

# 確保腳本在其所在目錄下執行
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
cd "$SCRIPT_DIR/.."

# Proto root directory
proto_root="pkg/proto"

# 檢查 proto 根目錄是否存在
if [ ! -d "$proto_root" ]; then
  echo "Error: Proto root directory '$proto_root' does not exist."
  exit 1
fi

# 找到所有 .proto 文件
proto_files=$(find "$proto_root" -name "*.proto")

if [ -z "$proto_files" ]; then
  echo "No .proto files found in the specified directory or its subdirectories."
  exit 0
fi

# 編譯每個 .proto 文件
for proto in $proto_files; do
  # 編譯 .proto 文件，直接輸出到與 .proto 文件相同的目錄
  protoc \
    --proto_path="$proto_root" \
    --go_out="$proto_root" \
    --go-grpc_out="$proto_root" \
    "$proto"
done

echo "All proto files have been compiled successfully!"