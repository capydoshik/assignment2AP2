package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"time"

	orderv1 "github.com/ayazb/ap2-generated-contracts/order/v1"
	paymentv1 "github.com/ayazb/ap2-generated-contracts/payment/v1"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"order-service/internal/clients"
	"order-service/internal/config"
	"order-service/internal/repository"
	grpcadapter "order-service/internal/transport/grpc"
	httpadapter "order-service/internal/transport/http"
	"order-service/internal/usecase"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatalf("failed to load .env: %v", err)
	}

	dbURL := config.MustGetEnv("ORDER_DB_URL")
	httpAddr := config.MustGetEnv("ORDER_HTTP_ADDR")
	grpcAddr := config.MustGetEnv("ORDER_GRPC_ADDR")
	paymentAddr := config.MustGetEnv("PAYMENT_GRPC_ADDR")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open order_db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to order_db: %v", err)
	}

	orderRepo := repository.NewOrderRepository(db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	paymentConn, err := grpcpkg.DialContext(
		ctx,
		paymentAddr,
		grpcpkg.WithTransportCredentials(insecure.NewCredentials()),
		grpcpkg.WithBlock(),
	)
	if err != nil {
		log.Fatalf("failed to connect to payment gRPC server: %v", err)
	}
	defer paymentConn.Close()

	paymentClient := clients.NewPaymentClientGRPC(paymentv1.NewPaymentServiceClient(paymentConn), 2*time.Second)
	orderUC := usecase.NewOrderUsecase(orderRepo, paymentClient)
	handler := httpadapter.NewHandler(orderUC)

	router := gin.Default()
	router.POST("/orders", handler.CreateOrder)
	router.GET("/orders/:id", handler.GetOrder)
	router.PATCH("/orders/:id/cancel", handler.CancelOrder)
	router.PATCH("/orders/:id/status", handler.UpdateOrderStatus)

	grpcServer := grpcpkg.NewServer()
	orderv1.RegisterOrderServiceServer(grpcServer, grpcadapter.NewOrderServer(orderUC, 500*time.Millisecond))

	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcAddr, err)
	}

	errCh := make(chan error, 2)

	go func() {
		log.Printf("Order Service REST running on %s", httpAddr)
		errCh <- router.Run(httpAddr)
	}()

	go func() {
		log.Printf("Order Service gRPC running on %s", grpcAddr)
		errCh <- grpcServer.Serve(listener)
	}()

	if err := <-errCh; err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
