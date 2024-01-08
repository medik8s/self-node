package peerhealth

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"

	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	selfNodeRemediationApis "github.com/medik8s/self-node-remediation/api"
	"github.com/medik8s/self-node-remediation/api/v1alpha1"
	"github.com/medik8s/self-node-remediation/controllers"
	"github.com/medik8s/self-node-remediation/pkg/certificates"
)

const (
	connectionTimeout = 5 * time.Second
	//IMPORTANT! this MUST be less than PeerRequestTimeout in apicheck
	//The difference between them should allow some time for sending the request over the network
	//todo enforce this
	apiServerTimeout = 3 * time.Second
)

var (
	snrRes = schema.GroupVersionResource{
		Group:    v1alpha1.GroupVersion.Group,
		Version:  v1alpha1.GroupVersion.Version,
		Resource: "selfnoderemediations",
	}
	nodeRes = schema.GroupVersionResource{
		Group:    corev1.SchemeGroupVersion.Group,
		Version:  corev1.SchemeGroupVersion.Version,
		Resource: "nodes",
	}
)

type Server struct {
	UnimplementedPeerHealthServer
	client     dynamic.Interface
	snr        *controllers.SelfNodeRemediationReconciler
	log        logr.Logger
	certReader certificates.CertStorageReader
	port       int
}

// NewServer returns a new Server
func NewServer(snr *controllers.SelfNodeRemediationReconciler, conf *rest.Config, log logr.Logger, port int, certReader certificates.CertStorageReader) (*Server, error) {

	// create dynamic client
	c, err := dynamic.NewForConfig(conf)
	if err != nil {
		return nil, err
	}

	return &Server{
		client:     c,
		snr:        snr,
		log:        log,
		certReader: certReader,
		port:       port,
	}, nil
}

// Start implements Runnable for usage by manager
func (s *Server) Start(ctx context.Context) error {

	serverCreds, err := certificates.GetServerCredentialsFromCerts(s.certReader)
	if err != nil {
		s.log.Error(err, "failed to get server credentials")
		return err
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		s.log.Error(err, "failed to listen")
		return err
	}

	opts := []grpc.ServerOption{
		grpc.ConnectionTimeout(connectionTimeout),
		grpc.Creds(serverCreds),
	}
	grpcServer := grpc.NewServer(opts...)
	RegisterPeerHealthServer(grpcServer, s)

	errChan := make(chan error)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- err
		}
	}()

	s.log.Info("peer health server started")

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		grpcServer.Stop()
	}
	return nil
}

// IsHealthy checks if the given node is healthy
func (s Server) IsHealthy(ctx context.Context, request *HealthRequest) (*HealthResponse, error) {

	nodeName := request.GetNodeName()
	if nodeName == "" {
		return nil, fmt.Errorf("empty node name in HealthRequest")
	}

	//fetch all snrs from all ns
	snrs := &v1alpha1.SelfNodeRemediationList{}
	if err := s.snr.List(ctx, snrs); err != nil {
		s.log.Error(err, "failed to fetch snrs")
		return nil, err
	}

	//return healthy only if all of snrs are considered healthy for that node
	for _, snr := range snrs.Items {
		if controllers.IsOwnedByNHC(&snr) {
			if healthCode := s.isHealthyBySnr(ctx, request.NodeName, snr.Namespace); healthCode != selfNodeRemediationApis.Healthy {
				return toResponse(healthCode)
			}
		} else if healthCode := s.isHealthyBySnr(ctx, request.MachineName, snr.Namespace); healthCode != selfNodeRemediationApis.Healthy {
			return toResponse(healthCode)
		}
	}
	return toResponse(selfNodeRemediationApis.Healthy)
}

func (s Server) isHealthyBySnr(ctx context.Context, snrName string, snrNamespace string) selfNodeRemediationApis.HealthCheckResponseCode {
	apiCtx, cancelFunc := context.WithTimeout(ctx, apiServerTimeout)
	defer cancelFunc()

	_, err := s.client.Resource(snrRes).Namespace(snrNamespace).Get(apiCtx, snrName, metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			s.log.Info("node is healthy")
			return selfNodeRemediationApis.Healthy
		}
		s.log.Error(err, "api error")
		return selfNodeRemediationApis.ApiError
	}

	s.log.Info("node is unhealthy")
	return selfNodeRemediationApis.Unhealthy
}

func (s Server) getNode(ctx context.Context, nodeName string) (*unstructured.Unstructured, error) {
	apiCtx, cancelFunc := context.WithTimeout(ctx, apiServerTimeout)
	defer cancelFunc()

	node, err := s.client.Resource(nodeRes).Namespace("").Get(apiCtx, nodeName, metav1.GetOptions{})
	if err != nil {
		s.log.Error(err, "api error")
		return nil, err
	}
	return node, nil
}

func toResponse(status selfNodeRemediationApis.HealthCheckResponseCode) (*HealthResponse, error) {
	return &HealthResponse{
		Status: int32(status),
	}, nil
}
