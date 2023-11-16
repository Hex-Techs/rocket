package clustertools

import (
	"fmt"

	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/utils/config"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

// createBasicKubeConfig creates a basic, general KubeConfig object that then can be extended
func createBasicKubeConfig(serverURL, clusterName, userName string, caCert []byte) *clientcmdapi.Config {
	// Use the cluster and the username as the context name
	contextName := fmt.Sprintf("%s@%s", userName, clusterName)

	var insecureSkipTLSVerify bool
	if caCert == nil {
		insecureSkipTLSVerify = true
	}

	return &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			clusterName: {
				Server:                   serverURL,
				InsecureSkipTLSVerify:    insecureSkipTLSVerify,
				CertificateAuthorityData: caCert,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userName,
			},
		},
		AuthInfos:      map[string]*clientcmdapi.AuthInfo{},
		CurrentContext: contextName,
	}
}

// CreateKubeConfigWithToken creates a KubeConfig object with access to the API server with a token
func CreateKubeConfigWithToken(serverURL, token string, caCert []byte) *clientcmdapi.Config {
	userName := "rocket-agent"
	clusterName := config.Read().Agent.Name
	config := createBasicKubeConfig(serverURL, clusterName, userName, caCert)
	config.AuthInfos[userName] = &clientcmdapi.AuthInfo{
		Token: token,
	}
	return config
}

// GenerateKubeConfigFromToken composes a kubeconfig from token
func GenerateKubeConfigFromToken(serverURL, token string, caCert []byte, flowRate int) (*rest.Config, error) {
	clientConfig := CreateKubeConfigWithToken(serverURL, token, caCert)
	config, err := clientcmd.NewDefaultClientConfig(*clientConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("error while creating kubeconfig: %v", err)
	}

	if flowRate < 0 {
		flowRate = 1
	}

	// here we magnify the default qps and burst in client-go
	config.QPS = rest.DefaultQPS * float32(flowRate)
	config.Burst = rest.DefaultBurst * flowRate

	return config, nil
}

func GenerateRestConfigFromCluster(cluster *cluster.Cluster) (*rest.Config, error) {
	configInternal := new(clientcmdapi.Config)
	if cluster.Kubeconfig != "" {
		c, err := clientcmd.Load([]byte(cluster.Kubeconfig))
		if err != nil {
			return nil, err
		}
		configInternal = c
	} else {
		configV1 := generateKubeConfig(cluster)
		configObject, err := clientcmdlatest.Scheme.ConvertToVersion(configV1, clientcmdapi.SchemeGroupVersion)
		if err != nil {
			return nil, err
		}
		configInternal = configObject.(*clientcmdapi.Config)
	}
	restConfig, err := clientcmd.NewDefaultClientConfig(*configInternal, &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmdapi.Cluster{Server: cluster.APIServer},
	}).ClientConfig()
	if err != nil {
		return nil, err
	}
	return restConfig, nil
}

func generateKubeConfig(cluster *cluster.Cluster) *clientcmdapiv1.Config {
	cfg := &clientcmdapiv1.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []clientcmdapiv1.NamedCluster{
			{
				Name: cluster.Name,
				Cluster: clientcmdapiv1.Cluster{
					Server:                   cluster.APIServer,
					CertificateAuthorityData: []byte(cluster.CAData),
				},
			},
		},
		AuthInfos: []clientcmdapiv1.NamedAuthInfo{
			{
				Name: cluster.Name,
				AuthInfo: clientcmdapiv1.AuthInfo{
					ClientCertificateData: []byte(cluster.CertData),
					ClientKeyData:         []byte(cluster.KeyData),
					Token:                 string(cluster.Token),
				},
			},
		},
		Contexts: []clientcmdapiv1.NamedContext{
			{
				Name: cluster.Name,
				Context: clientcmdapiv1.Context{
					Cluster:  cluster.Name,
					AuthInfo: cluster.Name,
				},
			},
		},
		CurrentContext: cluster.Name,
	}
	return cfg
}
