package main

import (
	"context"
	"fmt"
	"time"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
)

// detectChainID queries both the gRPC endpoint and the CometBFT RPC endpoint
// for the chain ID, validates that they agree, and returns the chain ID.
// Returns an error if either endpoint is unreachable or if they return
// different chain IDs.
func detectChainID(ctx context.Context, grpcAddr, nodeRPCAddr string) (string, error) {
	detectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	grpcChainID, err := chainIDFromGRPC(detectCtx, grpcAddr)
	if err != nil {
		return "", fmt.Errorf("failed to detect chain ID via gRPC (%s): %w", grpcAddr, err)
	}

	rpcChainID, err := chainIDFromRPC(detectCtx, nodeRPCAddr)
	if err != nil {
		return "", fmt.Errorf("failed to detect chain ID via node RPC (%s): %w", nodeRPCAddr, err)
	}

	if grpcChainID != rpcChainID {
		return "", fmt.Errorf("chain ID mismatch: gRPC returned %q, node RPC returned %q", grpcChainID, rpcChainID)
	}

	return grpcChainID, nil
}

func chainIDFromGRPC(ctx context.Context, grpcAddr string) (string, error) {
	conn, err := grpc.DialContext(ctx, grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	resp, err := cmtservice.NewServiceClient(conn).GetNodeInfo(ctx, &cmtservice.GetNodeInfoRequest{})
	if err != nil {
		return "", fmt.Errorf("GetNodeInfo: %w", err)
	}

	return resp.DefaultNodeInfo.Network, nil
}

func chainIDFromRPC(ctx context.Context, nodeRPCAddr string) (string, error) {
	rpcClient, err := rpchttp.New(nodeRPCAddr, "/websocket")
	if err != nil {
		return "", fmt.Errorf("create client: %w", err)
	}

	status, err := rpcClient.Status(ctx)
	if err != nil {
		return "", fmt.Errorf("status: %w", err)
	}

	return status.NodeInfo.Network, nil
}
