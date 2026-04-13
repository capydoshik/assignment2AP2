package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	orderv1 "github.com/ayazb/ap2-generated-contracts/order/v1"
	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("usage: go run ./cmd/order-stream-client <grpc-address> <order-id>")
	}

	address := os.Args[1]
	orderID := os.Args[2]

	conn, err := grpcpkg.Dial(address, grpcpkg.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to order gRPC server: %v", err)
	}
	defer conn.Close()

	client := orderv1.NewOrderServiceClient(conn)
	stream, err := client.SubscribeToOrderUpdates(context.Background(), &orderv1.OrderRequest{
		OrderId: orderID,
	})
	if err != nil {
		log.Fatalf("failed to subscribe: %v", err)
	}

	for {
		update, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Fatalf("stream error: %v", err)
		}

		fmt.Printf(
			"order=%s status=%s updated_at=%s\n",
			update.GetOrderId(),
			update.GetStatus(),
			update.GetUpdatedAt().AsTime().Format("2006-01-02 15:04:05"),
		)
	}
}
