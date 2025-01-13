package compose

import (
	"encoding/json"
	"log"
	"os/exec"
)

type ExpectedNetworkInspect struct {
	Name       string                 `json:"Name"`
	ID         string                 `json:"Id"`
	Labels     map[string]string      `json:"Labels"`
	Containers map[string]interface{} `json:"Containers"`
}

func AttachToDockerNetwork(networkName string, cid string) (bool, error) {
	// check if already attached
	networks := []ExpectedNetworkInspect{}
	out, err := doExec(exec.Command("docker", "network", "inspect", networkName, "--format", "json"))
	if err != nil {
		return false, nil
	}
	err = json.NewDecoder(out).Decode(&networks)
	if err != nil {
		return false, err
	}

	for _, n := range networks {
		if _, ok := n.Containers[cid]; ok {
			return false, nil
		}
	}

	// attach container to network
	log.Printf("[DEBUG] attaching container %s to network %s", cid, networkName)
	networkExec := exec.Command("docker", "network", "connect", networkName, cid)
	_, err = doExec(networkExec)
	if err != nil {
		return false, err
	}
	return true, nil
}
