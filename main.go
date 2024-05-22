package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	helpFlag := flag.Bool("h", false, "Prints this help message")
	versionFlag := flag.Bool("v", false, "Prints the version of the program")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("gofakeroot %s\n", version)
		os.Exit(0)
	}

	if *helpFlag {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Version: %s\n", version)
		os.Exit(0)
	}

	var cmd *exec.Cmd
	var err error
	if len(os.Args) < 2 {
		// start login shell of user
		cmd, err = findLoginShell(os.Getuid())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error finding login shell: %v\n", err)
			os.Exit(1)
		}
	} else {
		cmd = exec.Command(os.Args[1], os.Args[2:]...)

	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		},
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting command: %v\n", err)
		os.Exit(1)
	}

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Error waiting for command: %v\n", err)
		os.Exit(1)
	}
}

func findLoginShell(userid int) (*exec.Cmd, error) {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var loginShell string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		luid, shell, err := parsePasswdLine(scanner.Text())
		if err != nil {
			return nil, err
		}
		if luid == userid {
			loginShell = shell
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if loginShell == "" {
		return nil, errors.New("login shell not found for userid " + fmt.Sprint(userid))
	}

	cmd := exec.Command(loginShell)
	return cmd, nil
}

func parsePasswdLine(line string) (int, string, error) {
	sline := strings.Split(line, ":")
	if len(sline) < 7 {
		return 0, "", errors.New("invalid line in /etc/passwd")
	}
	uid, err := strconv.Atoi(sline[2])
	if err != nil {
		return 0, "", err
	}
	return uid, sline[6], nil
}
