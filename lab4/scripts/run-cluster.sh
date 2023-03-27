#!/bin/bash

# Small utility to run N instances of the KV server based on a shardmap
# Requires the JSON utility `jq` to be installed: https://stedolan.github.io/jq/

# Usage: run-cluster.sh shardmap.json
#
# Runs a process using server.go for each node in the shardmap in the background.

set -e

shardmap_file=$1
nodes=$(jq -r '.nodes | keys[]' < $shardmap_file)
for node in $nodes ; do
	go run cmd/server/server.go --shardmap "$shardmap_file" --node "$node" &
done

wait