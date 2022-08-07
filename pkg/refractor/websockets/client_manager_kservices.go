package websockets

import (
	"fmt"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func (cm *ClientManager) UpdateKService(kservice *servingv1.Service) error {
	panic(fmt.Errorf("todo ClientManager.UpdateKService"))

	// log.Printf("KService updated: %s", kservice.Name)

	// return nil
}

func (cm *ClientManager) DeleteKService(serviceName string) {
	panic(fmt.Errorf("todo ClientManager.DeleteKService"))

	// log.Printf("KService removed: %s", serviceName)
}
