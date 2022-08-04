package websockets

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func (cm *ClientManager) UpdateConfigMap(secret *corev1.ConfigMap) error {
	panic(fmt.Errorf("todo ClientManager.UpdateConfigMap"))

	// log.Printf("ConfigMap updated: %s", secret.Name)

	// return nil
}

func (cm *ClientManager) DeleteConfigMap(secretName string) {
	panic(fmt.Errorf("todo ClientManager.DeleteConfigMap"))

	// log.Printf("ConfigMap removed: %s", secretName)
}
