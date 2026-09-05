# lifecycle_exec.go Companion

`lifecycleWorkDir`：stop 配方遇缺失目录则 `cmd.Dir` 为空（P4 无 `go_relayToTrae/`）。

`startCommandDir`：启动路径不得把缺失 `working_dir` 交给 `exec`（Go 会报 `fork/exec /usr/bin/bash: no such file or directory`）。`./bin/` ELF 继承编排器 cwd；其它配方返回明确的 `working_dir %q does not exist`。
