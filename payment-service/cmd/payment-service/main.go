package main

import (
	"database/sql"
	"log"
	"net"

	_ "github.com/lib/pq"
	grpcpkg "google.golang.org/grpc"

	paymentv1 "github.com/ayazb/ap2-generated-contracts/payment/v1"

	"payment-service/internal/config"
	"payment-service/internal/repository"
	grpcadapter "payment-service/internal/transport/grpc"
	"payment-service/internal/usecase"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatalf("failed to load .env: %v", err)
	}

	dbURL := config.MustGetEnv("PAYMENT_DB_URL")
	grpcAddr := config.MustGetEnv("PAYMENT_GRPC_ADDR")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open payment_db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to payment_db: %v", err)
	}

	repo := repository.NewPaymentRepository(db)
	usecase := usecase.NewPaymentUsecase(repo)
	grpcServer := grpcpkg.NewServer(grpcpkg.UnaryInterceptor(grpcadapter.LoggingInterceptor))
	paymentv1.RegisterPaymentServiceServer(grpcServer, grpcadapter.NewPaymentServer(usecase))

	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcAddr, err)
	}

	log.Printf("Payment Service gRPC running on %s", grpcAddr)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to run payment gRPC server: %v", err)
	}
}
