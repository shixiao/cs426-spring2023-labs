package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"

	"cs426.yale.edu/lab4/kv"
	"cs426.yale.edu/lab4/kv/proto"
	"cs426.yale.edu/lab4/logging"
	"google.golang.org/grpc"
)

// Main entry-point for actually running your KV implementation as a single node.
//
// Takes in the shard map as a JSON file (see shardmaps/ for examples), and a nodeName.
//
// Listens on the port given for the node in the shardmap file automatically.
// You may want to start multiple nodes concurrently on your machine using different
// ports to test the cluster functionality. See scripts/run-cluster.sh for running
// a bunch of "nodes" as separate processes locally on your machine.

var (
	shardMapFile = flag.String("shardmap", "", "Path to a JSON file which describes the shard map")
	nodeName     = flag.String("node", "", "Name of the node (must match in shard map file)")
)

func main() {
	flag.Parse()
	logging.InitLogging()

	if len(*shardMapFile) == 0 || len(*nodeName) == 0 {
		logrus.Fatal("--shardmap and --node are required")
	}

	server := grpc.NewServer()
	fileSm, err := kv.WatchShardMapFile(*shardMapFile)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("loaded shardmap: %q", fileSm.ShardMap.Nodes())

	nodeInfo, ok := fileSm.ShardMap.Nodes()[*nodeName]
	if !ok {
		logrus.Fatalf("node not found in shard map: %s", *nodeName)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", nodeInfo.Port))
	if err != nil {
		logrus.Fatalf("failed to listen: %v", err)
	}

	clientPool := kv.MakeClientPool(&fileSm.ShardMap)

	proto.RegisterKvServer(
		server,
		kv.MakeKvServer(*nodeName, &fileSm.ShardMap, &clientPool),
	)
	logrus.Infof("server listening at %v", lis.Addr())
	if err := server.Serve(lis); err != nil {
		logrus.Fatalf("failed to serve: %v", err)
	}
}
