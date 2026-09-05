//go:build ignore

package main

import (
"fmt"
"os/exec"
)

func main() {
path, err := exec.LookPath("bash")
fmt.Printf("LookPath bash: %q, err: %v\n", path, err)

cmd := exec.Command(path, "-c", "echo hello")
out, err := cmd.CombinedOutput()
fmt.Printf("Command output: %q, err: %v\n", string(out), err)

cmd2 := exec.Command("/bin/bash", "-c", "echo hello2")
out2, err2 := cmd2.CombinedOutput()
fmt.Printf("Command /bin/bash output: %q, err: %v\n", string(out2), err2)
}
