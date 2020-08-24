package converge

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/flant/logboek"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"flant/deckhouse-candi/pkg/kubernetes/client"
	"flant/deckhouse-candi/pkg/util/retry"
)

func GetCloudConfig(kubeCl *client.KubernetesClient, nodeGroupName string) (string, error) {
	var cloudData string
	err := retry.StartLoop(fmt.Sprintf("Get %s cloud config️", nodeGroupName), 45, 5, func() error {
		secret, err := kubeCl.CoreV1().Secrets("d8-cloud-instance-manager").Get("manual-bootstrap-for-"+nodeGroupName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		cloudData = base64.StdEncoding.EncodeToString(secret.Data["cloud-config"])
		return nil
	})
	return cloudData, err
}

func CreateNodeGroup(kubeCl *client.KubernetesClient, nodeGroupName string, data map[string]interface{}) error {
	doc := unstructured.Unstructured{}
	doc.SetUnstructuredContent(data)

	resourceSchema := schema.GroupVersionResource{Group: "deckhouse.io", Version: "v1alpha1", Resource: "nodegroups"}

	return retry.StartLoop(fmt.Sprintf("Create NodeGroup %q", nodeGroupName), 45, 15, func() error {
		res, err := kubeCl.Dynamic().Resource(resourceSchema).Create(&doc, metav1.CreateOptions{})
		if err == nil {
			logboek.LogInfoF("NodeGroup %q created\n", res.GetName())
			return nil
		}

		if errors.IsAlreadyExists(err) {
			logboek.LogInfoF("Object %v, updating...", err)
			content, err := doc.MarshalJSON()
			if err != nil {
				return err
			}
			_, err = kubeCl.Dynamic().Resource(resourceSchema).Patch(doc.GetName(), types.MergePatchType, content, metav1.PatchOptions{})
			if err != nil {
				return err
			}
			logboek.LogInfoLn("OK!")
		}
		return nil
	})
}

func IsNodeExistsInCluster(kubeCl *client.KubernetesClient, nodeName string) (bool, error) {
	isExists := false
	err := retry.StartLoop(fmt.Sprintf("Checking that single Node %q exists", nodeName), 100, 20, func() error {
		_, err := kubeCl.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		isExists = true
		return nil
	})
	return isExists, err
}

func WaitForSingleNodeBecomeReady(kubeCl *client.KubernetesClient, nodeName string) error {
	return retry.StartLoop(fmt.Sprintf("Waiting for  Node %s to become Ready", nodeName), 100, 20, func() error {
		node, err := kubeCl.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		for _, c := range node.Status.Conditions {
			if c.Type == apiv1.NodeReady {
				if c.Status == apiv1.ConditionTrue {
					return nil
				}
			}
		}

		return fmt.Errorf("node %q is not Ready yet", nodeName)
	})
}

func WaitForNodesBecomeReady(kubeCl *client.KubernetesClient, nodeGroupName string, desiredReadyNodes int) error {
	return retry.StartLoop(fmt.Sprintf("Waiting for NodeGroup %s to become Ready", nodeGroupName), 100, 20, func() error {
		nodes, err := kubeCl.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: "node.deckhouse.io/group=" + nodeGroupName})
		if err != nil {
			return err
		}

		readyNodes := make(map[string]struct{})

		for _, node := range nodes.Items {
			for _, c := range node.Status.Conditions {
				if c.Type == apiv1.NodeReady {
					if c.Status == apiv1.ConditionTrue {
						readyNodes[node.Name] = struct{}{}
					}
				}
			}
		}

		message := fmt.Sprintf("Nodes Ready %v of %v\n", len(readyNodes), desiredReadyNodes)
		for _, node := range nodes.Items {
			condition := "NotReady"
			if _, ok := readyNodes[node.Name]; ok {
				condition = "Ready"
			}
			message += fmt.Sprintf("* %s | %s\n", node.Name, condition)
		}

		if len(readyNodes) >= desiredReadyNodes {
			logboek.LogInfoLn(message)
			return nil
		}

		return fmt.Errorf(strings.TrimSuffix(message, "\n"))
	})
}
