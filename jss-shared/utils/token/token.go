package token

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/metadata"
)

func BearerFromMD(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("missing authorization token")
	}
	vals := md["authorization"]
	if len(vals) == 0 {
		return "", fmt.Errorf("missing authorization token")
	}
	authHeader := vals[0]
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return "", fmt.Errorf("missing authorization token")
	}
	return strings.TrimSpace(authHeader[7:]), nil
}
