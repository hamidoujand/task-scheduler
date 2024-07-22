// docker is going to provide staring and stopping docker container
package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
)

// Container represents the info about the running container.
type Container struct {
	Id       string
	HostPort string
	Name     string
}

// StartContainer creates a container out of provided image and assign the name to that container and handles args.
func StartContainer(image string, name string, port string, dockerArgs []string, imageArgs []string) (Container, error) {
	args := []string{"run", "-P", "-d", "--name", name}
	//append docker args first
	args = append(args, dockerArgs...)
	//pass the image name
	args = append(args, image)
	//pass args related to image
	args = append(args, imageArgs...)

	var output bytes.Buffer
	command := exec.Command("docker", args...)
	//change the stdout
	command.Stdout = &output
	if err := command.Run(); err != nil {
		return Container{}, fmt.Errorf("start container for image %s: %w", image, err)
	}

	//get the id
	id := output.String()[:12]

	host, port, err := extractHostPort(id, port)
	if err != nil {
		return Container{}, fmt.Errorf("extract host/port: %w", err)
	}

	c := Container{
		Id:       id,
		HostPort: net.JoinHostPort(host, port),
		Name:     name,
	}

	return c, nil
}

// Stop is going to stop the current container and remove it as well.
func (c Container) Stop() error {
	command := exec.Command("docker", "stop", c.Id)

	if err := command.Run(); err != nil {
		return fmt.Errorf("stopping container %s: %w", c.Id, err)
	}

	//remove container
	command = exec.Command("docker", "rm", c.Id)
	if err := command.Run(); err != nil {
		return fmt.Errorf("removing container %s: %w", c.Id, err)
	}

	return nil
}

// DumpLogs is going to return the combined logs of the current container, from both stderr and stdout.
func (c Container) DumpLogs() []byte {
	logs, err := exec.Command("docker", "logs", c.Id).CombinedOutput()
	if err != nil {
		return nil
	}
	return logs
}

func extractHostPort(id string, port string) (ip string, boundedPort string, err error) {
	template := fmt.Sprintf("[{{range $k,$v := (index .NetworkSettings.Ports \"%s/tcp\")}}{{json $v}}{{end}}]", port)

	command := exec.Command("docker", "inspect", "-f", template, id)
	var output bytes.Buffer
	command.Stdout = &output

	if err := command.Run(); err != nil {
		return "", "", fmt.Errorf("inspect container %s: %w", id, err)
	}

	// Got  [{"HostIp":"0.0.0.0","HostPort":"49190"}{"HostIp":"::","HostPort":"49190"}]
	// Need [{"HostIp":"0.0.0.0","HostPort":"49190"},{"HostIp":"::","HostPort":"49190"}]
	data := bytes.ReplaceAll(output.Bytes(), []byte("}{"), []byte("},{"))

	var results []struct {
		HostIp   string
		HostPort string
	}

	if err := json.Unmarshal(data, &results); err != nil {
		return "", "", fmt.Errorf("unmarshal data: %w", err)
	}

	for _, result := range results {
		if result.HostIp != "::" {
			if result.HostIp == "" {
				//localhost
				return "localhost", result.HostPort, nil
			}
			return result.HostIp, result.HostPort, nil
		}
	}
	return "", "", fmt.Errorf("could not locate ip/port for container %s", id)
}
