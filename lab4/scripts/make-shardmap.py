#!/usr/bin/env python3

import argparse
import json
import random

parser = argparse.ArgumentParser(description='Generate shardmaps')
parser.add_argument('--nodes', type=int, required=True,
                    help='Number of nodes in the shardmap')
parser.add_argument('--shards', type=int, required=True,
                    help='Number of shards in the shardmap')
parser.add_argument('--replicas', type=int, default=1,
                    help='Number of replicas of each shard')
parser.add_argument('--max-replicas', type=int, default=None,
                    help='If set, replicas will vary randomly between --replicas[=1] and --max-replicas')
parser.add_argument('--stripe', action='store_true',
                    help='If set, stripe shards across nodes instead of randomly assigning for more fair balance')

parser.add_argument('--address', default='127.0.0.1',
                    help='IP Address for nodes')
parser.add_argument('--base-port', default=9000,
                    help='Starting port number for nodes')

def make_shardmap(args):
    nodes = {}
    for i in range(args.nodes):
        nodes[f"n{i+1}"] = {
            'address': args.address,
            'port': args.base_port + i
        }
    shards = {}
    counter = 0
    node_names = list(nodes.keys())
    for i in range(args.shards):
        num_replicas = args.replicas
        if args.max_replicas is not None:
            num_replicas = random.randint(num_replicas, args.max_replicas)
        shards[i + 1] = ["n1"]
        if args.stripe:
            nodes_for_shard = []
            for _ in range(num_replicas):
                nodes_for_shard.append(node_names[counter % len(node_names)])
                counter += 1
            shards[i + 1] = nodes_for_shard
        else:
            shards[i + 1] = random.sample(node_names, k=num_replicas)
    return {
        'numShards': args.shards,
        'nodes': nodes,
        'shards': shards,
    }


def main():
    args = parser.parse_args()
    shardmap = make_shardmap(args)
    print(json.dumps(shardmap, indent=4))


if __name__ == '__main__':
    main()
