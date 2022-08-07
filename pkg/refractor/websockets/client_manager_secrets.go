package websockets

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func (cm *ClientManager) UpdateSecret(secret *corev1.Secret) error {
	panic(fmt.Errorf("todo ClientManager.UpdateSecret"))

	// log.Printf("Secret updated: %s", secret.Name)

	// return nil
}

func (cm *ClientManager) DeleteSecret(secretName string) {
	panic(fmt.Errorf("todo ClientManager.DeleteSecret"))

	// log.Printf("Secret removed: %s", secretName)
}
