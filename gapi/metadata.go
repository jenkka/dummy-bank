package gapi

import (
	"context"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const (
	grpcGatewayUserAgentHeader = "grpcgateway-user-agent"
	xForwardedForHeader        = "x-forwarded-for"
	userAgentHeader            = "user-agent"
)

type Metadata struct {
	UserAgent string
	ClientIp  string
}

func (server *Server) ExtractMetadata(ctx context.Context) *Metadata {
	extractedMetadata := &Metadata{}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		userAgent := ""
		if userAgents := md.Get(grpcGatewayUserAgentHeader); len(userAgents) > 0 {
			userAgent = userAgents[0]
		} else if userAgents := md.Get(userAgentHeader); len(userAgents) > 0 {
			userAgent = userAgents[0]
		}
		extractedMetadata.UserAgent = userAgent

		if clientIps := md.Get(xForwardedForHeader); len(clientIps) > 0 {
			extractedMetadata.ClientIp = clientIps[0]
		}
	}

	if extractedMetadata.ClientIp == "" {
		if p, ok := peer.FromContext(ctx); ok {
			extractedMetadata.ClientIp = p.Addr.String()
		}
	}

	return extractedMetadata
}
