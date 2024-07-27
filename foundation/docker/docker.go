// docker is going to provide staring and stopping docker container
package docker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"time"
)

// Container represents the info about the running container.
type Container struct {
	Id       string
	HostPort string
}

// StartContainer creates a container out of provided image and assign the name to that container and handles args.
func StartContainer(image string, name string, port string, dockerArgs []string, imageArgs []string) (Container, error) {
	//we do 2 retries and another final one outside of the loop

	for i := range 2 {
		c, err := startContainer(image, name, port, dockerArgs, imageArgs)

		if err == nil {
			return c, nil
		}

		//wait few millis
		time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
	}

	return startContainer(image, name, port, dockerArgs, imageArgs)
}

// StartContainer creates a container out of provided image and assign the name to that container and handles args.
func startContainer(image string, name string, port string, dockerArgs []string, imageArgs []string) (Container, error) {

	//check to see if already a container with this NAME is running.
	if c, err := exists(name, port); err == nil {
		return c, nil
	}

	//The -P flag tells Docker to publish all exposed ports to random ports on the host.
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
		//also stop container
		stopContainer(id)
		return Container{}, fmt.Errorf("extract host/port: %w", err)
	}

	c := Container{
		Id:       id,
		HostPort: net.JoinHostPort(host, port),
	}

	return c, nil
}

// Stop is going to stop the current container and remove it as well.
func (c Container) Stop() error {
	return stopContainer(c.Id)
}

func stopContainer(id string) error {
	command := exec.Command("docker", "stop", id)

	if err := command.Run(); err != nil {
		return fmt.Errorf("stopping container %s: %w", id, err)
	}

	//remove container
	command = exec.Command("docker", "rm", id)
	if err := command.Run(); err != nil {
		return fmt.Errorf("removing container %s: %w", id, err)
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

// NetworkSetting represents the info required from "NetworkSetting" field inside of "docker inspect" result.
type NetworkSettings struct {
	Ports map[string][]struct {
		HostIP   string `json:"hostIp"`
		HostPort string `json:"HostPort"`
	} `json:"ports"`
}

// ContainerInfo is the type represents info we need from "docker inspect" command.
type ContainerInfo struct {
	NetworkSettings NetworkSettings `json:"NetworkSettings"`
}

func extractHostPort(id string, port string) (ip string, boundedPort string, err error) {

	command := exec.Command("docker", "inspect", id)
	var output bytes.Buffer
	command.Stdout = &output

	if err := command.Run(); err != nil {
		return "", "", fmt.Errorf("inspect container %s: %w", id, err)
	}

	var containerInfos []ContainerInfo
	if err := json.Unmarshal(output.Bytes(), &containerInfos); err != nil {
		return "", "", fmt.Errorf("unmarshal container infos: %w", err)
	}

	portKey := port + "/tcp"
	// since we inspecting by ID only 1 container will be back
	ports := containerInfos[0].NetworkSettings.Ports[portKey]

	for _, result := range ports {
		if result.HostIP != "::" {
			if result.HostIP == "" {
				//localhost
				return "localhost", result.HostPort, nil
			}
			return result.HostIP, result.HostPort, nil
		}
	}
	return "", "", fmt.Errorf("could not locate ip/port for container %s", id)
}

// exists check to see if there is already a container with this id running and returns it if there is.
func exists(id string, port string) (Container, error) {
	host, port, err := extractHostPort(id, port)

	if err != nil {
		return Container{}, errors.New("container is not running")
	}

	c := Container{
		Id:       id,
		HostPort: net.JoinHostPort(host, port),
	}
	return c, nil
}
