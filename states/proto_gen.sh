#!/bin/bash


# 确保我们在 states 目录下
cd "$(dirname "$0")"

# 为 commproto 生成代码
echo "Generating code for commproto..."
protoc --proto_path=. --gogofaster_out=plugins=grpc,Mcommproto/comm.proto=braid-scaffold/states/commproto:. commproto/*.proto

# 为 gameproto 生成代码
echo "Generating code for gameproto..."
protoc --proto_path=. --gogofaster_out=plugins=grpc,Mcommproto/comm.proto=braid-scaffold/states/commproto,Muser/user.proto=braid-scaffold/states/user:. gameproto/*.proto

# 为 user 生成代码
echo "Generating code for user..."
protoc --proto_path=. --gogofaster_out=plugins=grpc,Mcommproto/comm.proto=braid-scaffold/states/commproto:. user/*.proto

echo "Proto generation complete."