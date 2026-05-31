package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jenkka/dummy-bank/api"
	db "github.com/jenkka/dummy-bank/db/sqlc"
	"github.com/jenkka/dummy-bank/gapi"
	pb "github.com/jenkka/dummy-bank/pb/dummybank/v1"
	"github.com/jenkka/dummy-bank/util"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("Failed to load config file:", err)
	}

	conn, err := sql.Open(config.DbDriver, config.DbSource)
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}
	defer conn.Close()

	store := db.NewStore(conn)

	go runGatewayServer(config, store)
	runGRPCServer(config, store)
}

func runGRPCServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(store, config)
	if err != nil {
		log.Fatal("Failed to create gRPC server:", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterDummyBankServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal("Failed to create listener:", err)
	}

	log.Printf("Starting gRPC server on %s", config.GRPCServerAddress)
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal("Failed to start gRPC server:", err)
	}
}

func runGatewayServer(config util.Config, store db.Store) {
	server, err := gapi.NewServer(store, config)
	if err != nil {
		log.Fatal("Failed to create HTTP gateway server:", err)
	}

	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})

	grpcMux := runtime.NewServeMux(jsonOption)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = pb.RegisterDummyBankServiceHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatal("Failed to register handler server:", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", grpcMux)

	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatal("Failed to create listener:", err)
	}

	log.Printf("Starting HTTP gateway server on %s", config.GRPCServerAddress)
	err = http.Serve(listener, mux)
	if err != nil {
		log.Fatal("Failed to start HTTP gateway server:", err)
	}
}

func runGinServer(config util.Config, store db.Store) {
	server, err := api.NewServer(store, config)
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	log.Printf("Starting server on %s", config.HTTPServerAddress)
	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
