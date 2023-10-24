package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/kishorens18/grpc-mongodb/protobuf"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

type server struct {
	mongoClient *mongo.Client
	collection  *mongo.Collection
}

func (s *server) AddData(ctx context.Context, req *protobuf.DataRequest) (*protobuf.EmptyResponse, error) {
	// Marshal google.protobuf.Any to JSON
	jsonData, err := protoToJSON(req.Data)
	if err != nil {
		return nil, err
	}

	// Store JSON data in MongoDB
	_, err = s.collection.InsertOne(ctx, jsonData)
	if err != nil {
		return nil, err
	}

	return &protobuf.EmptyResponse{}, nil
}

func (s *server) GetData(ctx context.Context, req *protobuf.EmptyRequest) (*protobuf.DataResponse, error) {
	// Retrieve JSON data from MongoDB
	result := s.collection.FindOne(ctx, nil) // You need to add appropriate filters here
	if err := result.Err(); err != nil {
		return nil, err
	}

	// Unmarshal JSON data to google.protobuf.Any
	var jsonData map[string]interface{}
	err := result.Decode(&jsonData)
	if err != nil {
		return nil, err
	}

	anyData, err := jsonToProto(jsonData)
	if err != nil {
		return nil, err
	}

	return &protobuf.DataResponse{
		Data: anyData,
	}, nil
}

func protoToJSON(message proto.Message) (map[string]interface{}, error) {
	marshaler := jsonpb.Marshaler{}
	jsonString, err := marshaler.MarshalToString(message)
	if err != nil {
		return nil, err
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal([]byte(jsonString), &jsonData)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func jsonToProto(jsonData map[string]interface{}) (*protobuf.Data, error) {
	jsonString, err := json.Marshal(jsonData)
	if err != nil {
		return nil, err
	}

	var anyData protobuf.Data
	err = jsonpb.UnmarshalString(string(jsonString), &anyData)
	if err != nil {
		return nil, err
	}

	return &anyData, nil
}

func main() {
	// Set up MongoDB connection
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.TODO())

	// Set up gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	dataService := &server{
		mongoClient: client,
		collection:  client.Database("your-database-name").Collection("your-collection-name"),
	}
	protobuf.RegisterDataServiceServer(grpcServer, dataService)

	fmt.Println("gRPC server started on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
