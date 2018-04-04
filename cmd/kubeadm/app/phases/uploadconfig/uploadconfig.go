/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package uploadconfig

import (
	"fmt"

	"github.com/ghodss/yaml"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmapiext "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha1"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	kubeadmconfig "k8s.io/kubernetes/cmd/kubeadm/app/util/config"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

// UploadConfiguration saves the MasterConfiguration and MasterNodeConfiguration
// used for later reference (when upgrading for instance)
func UploadConfiguration(cfg *kubeadmapi.MasterConfiguration, client clientset.Interface) error {
	err := uploadMasterConfiguration(cfg, client)
	if err != nil {
		return err
	}

	return uploadMasterNodeConfiguration(cfg, client)
}

func uploadMasterConfiguration(cfg *kubeadmapi.MasterConfiguration, client clientset.Interface) error {
	fmt.Printf("[uploadconfig] Storing the configuration used in ConfigMap %q in the %q Namespace\n", kubeadmconstants.MasterConfigurationConfigMap, metav1.NamespaceSystem)

	// Convert cfg to the external version as that's the only version of the API that can be deserialized later
	externalcfg := &kubeadmapiext.MasterConfiguration{}
	legacyscheme.Scheme.Convert(cfg, externalcfg, nil)

	// Removes sensitive info from the data that will be stored in the config map
	externalcfg.Token = ""

	cfgYaml, err := yaml.Marshal(*externalcfg)
	if err != nil {
		return err
	}

	return apiclient.CreateOrUpdateConfigMap(client, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeadmconstants.MasterConfigurationConfigMap,
			Namespace: metav1.NamespaceSystem,
		},
		Data: map[string]string{
			kubeadmconstants.MasterConfigurationConfigMapKey: string(cfgYaml),
		},
	})
}

func uploadMasterNodeConfiguration(mainCfg *kubeadmapi.MasterConfiguration, client clientset.Interface) error {
	masterNodeConfigMapName, err := kubeadmconfig.GetMasterNodeConfigMapName()
	if err != nil {
		return err
	}

	fmt.Printf("[uploadconfig] Storing the master node configuration used in ConfigMap %q in the %q Namespace\n",
		masterNodeConfigMapName, metav1.NamespaceSystem)

	externalcfg := &kubeadmapiext.MasterNodeConfiguration{
		NodeName: mainCfg.NodeName,
	}

	cfgYaml, err := yaml.Marshal(*externalcfg)
	if err != nil {
		return err
	}

	node, err := client.CoreV1().Nodes().Get(mainCfg.NodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return apiclient.CreateOrUpdateConfigMap(client, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      masterNodeConfigMapName,
			Namespace: metav1.NamespaceSystem,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(node, v1.SchemeGroupVersion.WithKind("Node")),
			},
		},
		Data: map[string]string{
			kubeadmconstants.MasterNodeConfigurationConfigMapKey: string(cfgYaml),
		},
	})
}
