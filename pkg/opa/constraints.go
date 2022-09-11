package opa

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ConstraintMeta represents meta information of a constraint
type ConstraintMeta struct {
	Kind string
	Name string
}

// Violation represents each constraintViolation under status
type Violation struct {
	Kind              string `json:"kind"`
	Name              string `json:"name"`
	Namespace         string `json:"namespace,omitempty"`
	Message           string `json:"message"`
	EnforcementAction string `json:"enforcementAction"`
}

type ConstraintStatus struct {
	TotalViolations float64 `json:"totalViolations"`
	Violations      []*Violation
	AuditTimestamp  string `json:"audittimestamp"` //audittimestampadded
}

// type User struct {
// 	User string `json:"user"`
// }
type Cluster struct {
	ClusterName   string `json:"clustername"`
	ClusterType   string `json:"clustertype`
	CloudType     string `json:"cloudtype"`
	ClusterId     string `json:"clusterid"`
	ClusterRegion string `json:"clusterregion"`
	Assetid       string `json:"assetid"`
}

type ClusDet struct {
	Userid     string `json:"user_id"`
	Cluster    Cluster
	Kubechecks []Constraint `json:"kubechecks"`
}
type ClusVio struct {
	Info      ClusDet
	Violation Constraint
}

// Constraint
type Constraint struct {
	Meta   ConstraintMeta
	Spec   ConstraintSpec
	Status ConstraintStatus
}

// ConstraintSpec collect general information about the overall constraints applied to the cluster
type ConstraintSpec struct {
	EnforcementAction string `json:"enforcementAction"`
}

const (
	constraintsGV           = "constraints.gatekeeper.sh/v1beta1"
	constraintsGroup        = "constraints.gatekeeper.sh"
	constraintsGroupVersion = "v1beta1"
)

func createConfig(inCluster *bool) (*restclient.Config, error) {
	if *inCluster {
		log.Println("Using incluster K8S client")
		return restclient.InClusterConfig()
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Println("Could not find user HomeDir" + err.Error())
			return nil, err
		}

		kubeconfig := filepath.Join(home, ".kube", "config")

		// use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return config, nil
	}
}

func createKubeClient(inCluster *bool) (*kubernetes.Clientset, error) {

	config, err := createConfig(inCluster)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)

	if err != nil {
		log.Println(err)
		return nil, err
	}
	return clientset, nil
}

func GetClusterInfo(inCluster *bool) string {
	cfg, _ := createConfig(inCluster)
	link := cfg.Host + cfg.APIPath
	return link
}

func createKubeClientGroupVersion(inCluster *bool) (controllerClient.Client, error) {
	config, err := createConfig(inCluster)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	client, err := controllerClient.New(config, controllerClient.Options{})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return client, nil
}

// GetConstraints returns a list of all OPA constraints
func GetConstraints(inCluster *bool) ([]interface{}, error) {
	//	os.Setenv("CLOUD_TYPE", "AWS")
	//	os.Setenv("CLUSTER_REGION", "us-east-1")
	//	os.Setenv("CLUSTER_TYPE", "EKS")
	//	os.Setenv("CLUSTER_NAME", "dev-compliance-169aC2418")
	//	os.Setenv("CLUSTER_ID", "BBAAC95676A159BC46B8292577CE1D59")

	var clusterType string
	var clusterRegion string
	var cloudType string
	var clusterName string
	var clusterID string
	//	var assetID string
	var userID string

	cloudType = os.Getenv("CLOUD_TYPE")
	clusterRegion = os.Getenv("CLUSTER_REGION")
	clusterType = os.Getenv("CLUSTER_TYPE")
	clusterName = os.Getenv("CLUSTER_NAME")
	clusterID = os.Getenv("CLUSTER_ID")
	//	assetID = os.Getenv("ASSET_ID")
	userID = os.Getenv("USER_ID")

	client, err := createKubeClient(inCluster)
	if err != nil {
		return nil, err
	}

	cClient, err := createKubeClientGroupVersion(inCluster)
	if err != nil {
		return nil, err
	}

	c, err := client.ServerResourcesForGroupVersion(constraintsGV)
	if err != nil {
		return nil, err
	}

	var constraints []Constraint
	var clusdet []ClusDet
	for _, r := range c.APIResources {
		canList := false
		for _, verb := range r.Verbs {
			if verb == "list" {
				canList = true
				break
			}
		}

		if !canList {
			continue
		}
		actual := &unstructured.UnstructuredList{}
		actual.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   constraintsGroup,
			Kind:    r.Kind,
			Version: constraintsGroupVersion,
		})

		err = cClient.List(context.TODO(), actual)
		if err != nil {
			return nil, err
		}

		if len(actual.Items) > 0 {
			for _, item := range actual.Items {
				// kind := item.GetKind()
				// name := item.GetName()
				// namespace := item.GetNamespace()
				// log.Printf("Kind:%s, Name:%s, Namespace:%s \n", kind, name, namespace)
				var obj = item.Object
				var constraint Constraint
				data, err := json.Marshal(obj)
				if err != nil {
					log.Println(err)
					continue
				}
				err = json.Unmarshal(data, &constraint)
				if err != nil {
					log.Println(err)
					continue
				}
				//c := "Cluster Name: minikube-sathya"
				//constraints = append(c)

				constraints = append(constraints, Constraint{
					Meta:   ConstraintMeta{Kind: item.GetKind(), Name: item.GetName()},
					Status: ConstraintStatus{TotalViolations: constraint.Status.TotalViolations, Violations: constraint.Status.Violations, AuditTimestamp: constraint.Status.AuditTimestamp}, //audittimestampadded
					Spec:   ConstraintSpec{EnforcementAction: constraint.Spec.EnforcementAction},
				})

			}

		}

	}
	x := GetClusterInfo(inCluster)
	y := "/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy"

	clusdet = append(clusdet, ClusDet{
		Userid:     userID,
		Cluster:    Cluster{CloudType: cloudType, ClusterRegion: clusterRegion, ClusterType: clusterType, ClusterName: clusterName, ClusterId: clusterID, Assetid: x + y},
		Kubechecks: constraints})
	a := []interface{}{clusdet}
	//log.Print(a)
	return a, nil
}
