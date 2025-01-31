package server

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"k8s.io/klog/v2"

	"github.com/kubernetes-sigs/maintainers/experimental/ksandbox/pkg/client"
	protocol "github.com/kubernetes-sigs/maintainers/experimental/ksandbox/protocol/ksandbox/v1alpha1"
)

// AgentServer manages a GRPC server that includes our BuildAgent service
type AgentServer struct {
	grpcServer *grpc.Server

	protocol.UnimplementedAgentServer
}

var _ protocol.AgentServer = &AgentServer{}

// NewAgentServer constructs a AgentServer
func NewAgentServer() (*AgentServer, error) {
	s := &AgentServer{}
	return s, nil
}

// ListenAndServe runs the GRPC server on the specified endpoint
func (s *AgentServer) ListenAndServe(listen string, serverTLS credentials.TransportCredentials) error {
	var options []grpc.ServerOption
	if serverTLS != nil {
		options = append(options, grpc.Creds(serverTLS))
	}
	lis, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("failed to listen on %q: %w", listen, err)
	}

	s.grpcServer = grpc.NewServer(options...)
	protocol.RegisterAgentServer(s.grpcServer, s)
	return s.grpcServer.Serve(lis)
}

// assertPeerIsVerified is a sanity check that the peer has a verified certificate matching the expected CN
// In practice this should be enforced by the TLS layer (RequireAndVerifyClientCert).
// Because we use ephemeral PKI infrastrcture, there should only be one verified cert.
// Hence we panic if anything doesn't look right.
func assertPeerIsVerified(ctx context.Context) {
	clientInfo, ok := peer.FromContext(ctx)
	if !ok {
		klog.Fatalf("unable to get peer info")
	}
	authInfo := clientInfo.AuthInfo
	if authInfo == nil {
		klog.Fatalf("peer did not have auth info")
	}
	tlsInfo, ok := authInfo.(credentials.TLSInfo)
	if !ok {
		klog.Fatalf("authInfo was of unexpected type %T", authInfo)
	}
	for _, verifiedChain := range tlsInfo.State.VerifiedChains {
		for _, cert := range verifiedChain {
			for _, dnsName := range cert.DNSNames {
				if dnsName == client.ClientCertificateDNSName {
					return
				}
			}
		}
	}

	klog.Fatalf("did not find verified tls chain with expected CN")
}

// ExcecuteCommand executes the requested command and returns the results
func (s *AgentServer) ExecuteCommand(ctx context.Context, request *protocol.ExecuteCommandRequest) (*protocol.ExecuteCommandResponse, error) {
	if len(request.Command) == 0 {
		return nil, fmt.Errorf("Command is required")
	}

	assertPeerIsVerified(ctx)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.Command(request.Command[0], request.Command[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if request.WorkingDir != "" {
		cmd.Dir = request.WorkingDir
	}

	{
		env := make(map[string]string)
		environ := os.Environ()
		for _, kv := range environ {
			tokens := strings.Split(kv, "=")
			// TODO: Filter any env vars?
			env[tokens[0]] = kv
		}

		for _, ev := range request.Env {
			env[ev.Name] = ev.Name + "=" + ev.Value
		}

		for _, kv := range env {
			cmd.Env = append(cmd.Env, kv)
		}
	}

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to run command: %w", err)
		}
	}

	return &protocol.ExecuteCommandResponse{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: int32(exitCode),
	}, nil
}
