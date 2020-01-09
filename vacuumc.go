package main

import (
	"bytes"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"regexp"
	"strings"
	"time"
)

// global variables used across functions
var ctx context.Context
var cli client.APIClient

// desktopCheck: remove inactive desktop containers
func containerCheck(ctx context.Context, cli client.Client, container string, command []string, containerGrep string) {

	// creating command struct to run in container
	containerCommand, err := cli.ContainerExecCreate(
		ctx,
		container,
		types.ExecConfig{
			Tty:          bool(true),
			AttachStderr: bool(true),
			AttachStdout: bool(true),
			AttachStdin:  bool(true),
			Cmd:          command,
		},
	)
	if err != nil {
		panic(err)
	}
	execID := containerCommand.ID

	// execute command in container
	response, err := cli.ContainerExecAttach(
		ctx,
		execID,
		types.ExecStartCheck{Tty: bool(true), Detach: bool(false)},
	)
	if err != nil {
		panic(err)
	}

	// read from tty into buffer
	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Reader)

	// create string from bytes buffer
	ttyOutput := buf.String()
	response.Close()
	// fmt.Println("tty output:", "\n", ttyOutput)

	// if not "Xorg" / "sshd: " in output: remove container
	if !(strings.Contains(ttyOutput, containerGrep) == true) {
		fmt.Println("removing: ", container)
		cli.ContainerRemove(ctx, container, types.ContainerRemoveOptions{Force: bool(true)})
	} else {
		fmt.Println("Not removing: ", container)
	}
}

// vacuumStart: vacuumc checks running desktop containers and application containers for activity.
// If older than 1 minute and not active, the container should be killed
func vacuumStart() {

	// commands
	desktopCommand := []string{"ps", "-e"}
	desktopGrep := "Xorg"
	appCommand := []string{"ps", "aux"}
	appGrep := "sshd:"

	// context
	ctx := context.Background()

	// docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	// list all containers
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	// loop containers
	for _, container := range containers {

		// make smaller
		cont := strings.Join(container.Names, " ")[1:]

		// check if 'vcd-' in container name
		if strings.Contains(cont, "vcd-") == true {

			// check if container is up more than a minute
			if !(strings.Contains(container.Status, "seconds") == true) {

				// desktop or application?
				findSign := regexp.MustCompile("-")
				matches := findSign.FindAllStringIndex(cont, -1)
				lenMatches := len(matches)

				// printing containername that needs checking
				fmt.Println("checking: ", cont)

				// desktop check
				if lenMatches == 1 {
					containerCheck(ctx, *cli, cont, desktopCommand, desktopGrep)
				}

				// application check
				if lenMatches == 2 {
					containerCheck(ctx, *cli, cont, appCommand, appGrep)
				}
			}
		}
	}
}

// main function to run vacuumc, runs every 30 seconds
func main() {
	for {
		vacuumStart()
		time.Sleep(30 * time.Second)
	}
}
