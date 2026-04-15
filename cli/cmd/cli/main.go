package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	marketv1 "k8s-manager/proto/gen/v1/market"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var accessToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IkVFSElmdWVaVk52QURHY0l6cDZQaiJ9.eyJpc3MiOiJodHRwczovL2Rldi1uNzV2Yng0aGdkazRrcmFpLmV1LmF1dGgwLmNvbS8iLCJzdWIiOiJnb29nbGUtb2F1dGgyfDExMzY2OTE0NjIwMTc1NzE2NzkzMSIsImF1ZCI6WyJodHRwczovL2s4cy1iYWNrZW5kIiwiaHR0cHM6Ly9kZXYtbjc1dmJ4NGhnZGs0a3JhaS5ldS5hdXRoMC5jb20vdXNlcmluZm8iXSwiaWF0IjoxNzc2MTcwNDk0LCJleHAiOjE3NzYyNTY4OTQsInNjb3BlIjoib3BlbmlkIHByb2ZpbGUgZW1haWwiLCJhenAiOiJqUzRRM0RDSmxIbGttOWRqZGt4YjBHaEozREQ0d0E2WCJ9.jkMsQuPnB92FlfOHPt0dCfv2xNNxzCA3ZI1HNwSUFdRB-0dxh-U11jd3hBDeGpttbt_doVzZm5R9mJMpzQ_dpVyvdZD1C7TXhB5uNu-djw9GszilOwBf1BXlA9j36n-168IOnWjmM5oMcHfXq2SS2U_PoxmEwhH1jgRzYJOhO-prKXpDz6Qmmbj4ch0pESwRQgXuehmEFl9jCdctntNfEIwN3aAzEk52szkuogi4RD8UYv422RKa0xQceYp5JFgpI0ufjVkdRkZ1qkNDPIPRDXEcI5gfOroRb9NaCtdAyr99pZVheFA-SrDyjkC4YHIEklCqDO2iiRX1n-Tl2lb4yw"

func mustProtoJSON(m interface{ ProtoReflect() protoreflect.Message }) string {
	b, err := protojson.MarshalOptions{
		Multiline:       true,
		Indent:          "  ",
		UseProtoNames:   true,
		EmitUnpopulated: true,
	}.Marshal(m)
	if err != nil {
		return fmt.Sprintf("marshal error: %v", err)
	}
	return string(b)
}

func withAuth(ctx context.Context, accessToken string) context.Context {
	return metadata.AppendToOutgoingContext(
		ctx,
		"authorization", "Bearer "+accessToken,
	)
}

func main() {
	flag.StringVar(&accessToken, "access-token", accessToken, "Access Token")
	flag.Parse()

	ctx := withAuth(context.Background(), accessToken)

	conn, err := grpc.NewClient(
		"localhost:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := marketv1.NewPluginServiceClient(conn)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := client.ListPlugins(ctx, &marketv1.ListPluginsRequest{})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(mustProtoJSON(resp))
}
