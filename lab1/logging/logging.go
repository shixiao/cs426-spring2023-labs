package failure_injection

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

var (
	influxDbUrl       = flag.String("influx-url", "", "If set, URL of InfluxDB to log to")
	influxDbAuthToken = flag.String("influx-token", "", "Auth token for InfluxDB")
	influxOrg         = flag.String("influx-org", "influxdata", "InfluxDB org")
	influxBucket      = flag.String("influx-bucket", "default", "InfluxDB bucket")
	influxMeasurement = flag.String("influx-measurement", "", "InfluxDB measurement (dataset)")
)

type Logger interface {
	LogRequest(context context.Context, isError bool, latencyMs float64)
}

func logToStderr(peerAddr string, isError bool, latencyMs float64) {
	log.Printf("processed request from %s: isError=%t, latencyMs=%f", peerAddr, isError, latencyMs)
}

func getIpFromAddr(addr net.Addr) string {
	switch a := addr.(type) {
	case *net.TCPAddr:
		return a.IP.String()
	case *net.UDPAddr:
		return a.IP.String()
	}
	return "unknown"
}

func getPeerHost(context context.Context) string {
	peer, ok := peer.FromContext(context)
	peerAddr := "unknown"
	if ok {
		ip := getIpFromAddr(peer.Addr)
		hostname, err := net.LookupAddr(ip)
		if err == nil && len(hostname) >= 1 {
			peerAddr = hostname[0]
		} else {
			peerAddr = ip
		}
	}
	return peerAddr
}

type InfluxLogger struct {
	measurement string
	writeApi    api.WriteAPI
}

func MakeInfluxLogger(
	url string,
	authToken string,
	org string,
	bucket string,
	measurement string,
) *InfluxLogger {
	client := influxdb2.NewClient(url, authToken)
	writeApi := client.WriteAPI(org, bucket)
	return &InfluxLogger{
		measurement: measurement,
		writeApi:    writeApi,
	}
}

func (logger *InfluxLogger) LogRequest(context context.Context, isError bool, latencyMs float64) {
	peerAddr := getPeerHost(context)
	hostname, _ := os.Hostname()
	logToStderr(
		peerAddr,
		isError,
		latencyMs,
	)

	logger.writeApi.WritePoint(
		influxdb2.NewPoint(
			logger.measurement,
			map[string]string{
				"client": peerAddr,
				"host":   hostname,
			},
			map[string]interface{}{
				"is_error":   isError,
				"latency_ms": latencyMs,
			},
			time.Now(),
		),
	)
}

type StderrLogger struct{}

func (logger *StderrLogger) LogRequest(context context.Context, isError bool, latencyMs float64) {
	logToStderr(
		getPeerHost(context),
		isError,
		latencyMs,
	)
}

func MakeLogger() Logger {
	if len(*influxDbUrl) == 0 {
		return &StderrLogger{}
	}

	if len(*influxDbAuthToken) == 0 || len(*influxMeasurement) == 0 {
		log.Fatalf("Must set --influx-token and --influx-measurement")
	}
	return MakeInfluxLogger(*influxDbUrl, *influxDbAuthToken, *influxOrg, *influxBucket, *influxMeasurement)
}

func MakeMiddleware(logger Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now().UnixMicro()
		resp, err := handler(ctx, req)
		latency := float64(time.Now().UnixMicro()-startTime) / 1000.0

		logger.LogRequest(ctx, err != nil, latency)
		return resp, err
	}
}
