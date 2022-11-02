package controlplane

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-ping/ping"

	corev1 "k8s.io/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/medik8s/self-node-remediation/pkg/peers"
)

const (
	kubeletPort = "10250"
)

//Manager contains logic and info needed to fence and remediate controlplane nodes
type Manager struct {
	nodeName                     string
	nodeRole                     peers.Role
	endpointHealthCheckUrl       string
	wasEndpointAccessibleAtStart bool
	client                       client.Client
	log                          logr.Logger
}

//NewManager inits a new Manager return nil if init fails
func NewManager(nodeName string, myClient client.Client) *Manager {
	return &Manager{
		nodeName:                     nodeName,
		endpointHealthCheckUrl:       os.Getenv("END_POINT_HEALTH_CHECK_URL"),
		client:                       myClient,
		wasEndpointAccessibleAtStart: false,
		log:                          ctrl.Log.WithName("controlPlane").WithName("Manager"),
	}
}

func (manager *Manager) Start(ctx context.Context) error {
	if err := manager.initializeManager(); err != nil {
		return err
	}
	return nil
}

func (manager *Manager) IsControlPlane() bool {
	return manager.nodeRole == peers.ControlPlane
}

func (manager *Manager) IsControlPlaneHealthy(workerPeerResponse peers.Response, canOtherMastersBeReached bool) bool {
	switch workerPeerResponse.Reason {
	//reported unhealthy by worker peers
	case peers.UnHealthyBecausePeersResponse:
		return false
	case peers.UnHealthyBecauseNodeIsIsolated:
		return canOtherMastersBeReached
	//reported healthy by worker peers
	case peers.HealthyBecauseErrorsThresholdNotReached, peers.HealthyBecauseCRNotFound:
		return true
	//controlplane node has connection to most workers, we assume it's not isolated (or at least that the controlplane node that does not have worker peers quorum will reboot)
	case peers.HealthyBecauseMostPeersCantAccessAPIServer:
		return manager.isDiagnosticsPassed()
	case peers.HealthyBecauseNoPeersWereFound:
		return manager.isDiagnosticsPassed() && canOtherMastersBeReached

	default:
		errorText := "node is considered unhealthy by worker peers for an unknown reason"
		manager.log.Error(errors.New(errorText), errorText, "reason", workerPeerResponse.Reason, "node name", manager.nodeName)
		return false
	}

}

func (manager *Manager) isDiagnosticsPassed() bool {
	if manager.isEndpointAccessLost() {
		return false
	} else if !manager.isKubeletServiceRunning() {
		return false
	} else if !manager.isEtcdRunning() {
		return false
	}
	return true
}

func wrapWithInitError(err error) error {
	return fmt.Errorf("error initializing controlplane handler [%w]", err)
}

func (manager *Manager) initializeManager() error {

	node := corev1.Node{}
	key := client.ObjectKey{
		Name: manager.nodeName,
	}

	if err := manager.client.Get(context.TODO(), key, &node); err != nil {
		manager.log.Error(err, "could not retrieve node")
		return wrapWithInitError(err)
	}
	manager.setNodeRole(node)

	manager.wasEndpointAccessibleAtStart = manager.isEndpointAccessible()
	return nil
}

func (manager *Manager) setNodeRole(node corev1.Node) {
	if _, isWorker := node.Labels[peers.WorkerLabelName]; isWorker {
		manager.nodeRole = peers.Worker
	} else {
		peers.SetControlPlaneLabelType(&node)
		if _, isMaster := node.Labels[peers.GetUsedControlPlaneLabel()]; isMaster {
			manager.nodeRole = peers.ControlPlane
		}
	}
}

func (manager *Manager) isEndpointAccessLost() bool {
	if !manager.wasEndpointAccessibleAtStart {
		return false
	}
	return !manager.isEndpointAccessible()
}

func (manager *Manager) isEndpointAccessible() bool {
	if len(manager.endpointHealthCheckUrl) == 0 {
		return true
	}

	pinger, err := ping.NewPinger(manager.endpointHealthCheckUrl)
	if err != nil {
		manager.log.Error(err, "could not access endpoint", "endpoint URL", manager.endpointHealthCheckUrl)
		return false
	}
	pinger.Count = 3
	pinger.Timeout = time.Second * 5

	if err := pinger.Run(); err != nil {
		manager.log.Error(err, "could not access endpoint", "endpoint URL", manager.endpointHealthCheckUrl)
		return false
	}
	return true
}

func (manager *Manager) isKubeletServiceRunning() bool {
	url := fmt.Sprintf("https://%s:%s/pods", manager.nodeName, kubeletPort)
	cmd := exec.Command("curl", "-k", "-X", "GET", url)
	if err := cmd.Run(); err != nil {
		manager.log.Error(err, "kubelet service is down", "node name", manager.nodeName)

		return false
	}

	return true
}

func (manager *Manager) isEtcdRunning() bool {
	//TODO mshitrit implement
	return true
}