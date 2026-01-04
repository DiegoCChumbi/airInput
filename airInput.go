package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	osType := runtime.GOOS
	var pythonCmdName string
	var pythonScript string

	switch osType {
	case "windows":
		pythonCmdName = "python"
		pythonScript = "controller-win.py"
	case "linux":
		pythonCmdName = "python3"
		pythonScript = "controller-linux.py"
	default:
		log.Fatalf("Unsupported operating system: %s", osType)
	}

	if _, err := os.Stat("server.js"); os.IsNotExist(err) {
		log.Fatal("server.js no found")
	}
	if _, err := os.Stat(pythonScript); os.IsNotExist(err) {
		log.Fatalf("%s no found", pythonScript)
	}

	nodeCmd := exec.Command("node", "server.js")
	nodeCmd.Stdout = os.Stdout
	nodeCmd.Stderr = os.Stderr

	if err := nodeCmd.Start(); err != nil {
		log.Fatalf("Error starting node: %v", err)
	}

	time.Sleep(1 * time.Second)

	pyCmd := exec.Command(pythonCmdName, pythonScript)
	// pyCmd.Stdout = os.Stdout
	pyCmd.Stderr = os.Stderr

	if err := pyCmd.Start(); err != nil {
		nodeCmd.Process.Kill()
		log.Fatalf("Error starting Python: %v", err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println("Bye!")

	if nodeCmd.Process != nil {
		nodeCmd.Process.Kill()
	}
	if pyCmd.Process != nil {
		pyCmd.Process.Kill()
	}
}
