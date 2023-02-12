package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const debugPodName = "debug-pod"

// GetBootTime gets the boot time of the given node by running a pod on it executing uptime command
func GetBootTime(c *kubernetes.Clientset, nodeName string, ns string) (*time.Time, error) {
	output, err := RunCommandInCluster(c, nodeName, ns, []string{"uptime", "-s"})
	if err != nil {
		return nil, err
	}

	bootTime, err := time.Parse("2006-01-02 15:04:05", output)
	if err != nil {
		return nil, err
	}

	return &bootTime, nil
}

// RunCommandInCluster runs a command in a pod in the cluster and returns the output
func RunCommandInCluster(c *kubernetes.Clientset, nodeName string, ns string, command []string) (string, error) {
	// create a pod and wait that it's running
	podSpec := getPod(nodeName)
	var pod *corev1.Pod
	var err error
	Eventually(func() error {
		pod, err = c.CoreV1().Pods(ns).Create(context.Background(), podSpec, metav1.CreateOptions{})
		return err

	}, 6*time.Minute, 10*time.Second).ShouldNot(HaveOccurred())

	defer cleanUpDebugPod(c, nodeName, ns)

	err = waitForCondition(c, pod, corev1.PodReady, corev1.ConditionTrue, time.Minute)
	if err != nil {
		return "", err
	}

	logger.Info("helper pod is running, going to execute command", "command", command)
	//cmd := []string{"sh", "-c", fmt.Sprintf("microdnf install procps -y >/dev/null 2>&1 && %s", command)}
	outputBytes, err := waitForPodOutput(c, pod /*cmd*/, command)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(outputBytes)), nil
}

func cleanUpDebugPod(c *kubernetes.Clientset, nodeName string, ns string) {
	err := c.CoreV1().Pods(ns).Delete(context.Background(), generateDebugPodName(nodeName), metav1.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())
	Eventually(func() bool {
		_, err := c.CoreV1().Pods(ns).Get(context.Background(), nodeName, metav1.GetOptions{})
		return errors.IsNotFound(err)

	}, 6*time.Minute, 10*time.Second).Should(BeTrue())
	time.Sleep(30 * time.Second)
}

func waitForPodOutput(c *kubernetes.Clientset, pod *corev1.Pod, command []string) ([]byte, error) {
	var out []byte
	if err := wait.PollImmediate(15*time.Second, 5*time.Minute, func() (done bool, err error) {
		out, err = execCommandOnPod(c, pod, command)
		if err != nil {
			return false, err
		}

		return len(out) != 0, nil
	}); err != nil {
		return nil, err
	}

	return out, nil
}

// execCommandOnPod runs command in the pod and returns buffer output
func execCommandOnPod(c *kubernetes.Clientset, pod *corev1.Pod, command []string) ([]byte, error) {
	var outputBuf bytes.Buffer
	var errorBuf bytes.Buffer

	req := c.CoreV1().RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: pod.Spec.Containers[0].Name,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: &outputBuf,
		Stderr: &errorBuf,
		Tty:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to run command %v: error: %v, outputStream %s; errorStream %s", command, err, outputBuf.String(), errorBuf.String())
	}

	if errorBuf.Len() != 0 {
		return nil, fmt.Errorf("failed to run command %v: output %s; error %s", command, outputBuf.String(), errorBuf.String())
	}

	return outputBuf.Bytes(), nil
}

// waitForCondition waits until the pod will have specified condition type with the expected status
func waitForCondition(c *kubernetes.Clientset, pod *corev1.Pod, conditionType corev1.PodConditionType, conditionStatus corev1.ConditionStatus, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		updatedPod := &corev1.Pod{}
		var err error
		if updatedPod, err = c.CoreV1().Pods(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{}); err != nil {
			return false, nil
		}
		for _, c := range updatedPod.Status.Conditions {
			if c.Type == conditionType && c.Status == conditionStatus {
				return true, nil
			}
		}
		return false, nil
	})
}

func generateDebugPodName(nodeName string) string {
	return fmt.Sprintf("%s.%s", nodeName, debugPodName)
}

func getPod(nodeName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: generateDebugPodName(nodeName),
		},
		Spec: corev1.PodSpec{
			NodeName:      nodeName,
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    debugPodName,
					Image:   "ubuntu",
					Command: []string{"/bin/bash", "-ec", "while :; do echo '.'; sleep 5 ; done"},
				},
			},
		},
	}
}
