package main

import "fmt"

type ServiceNode struct {
	Service    Service
	DependsOn  []string
	Dependents []string
	InDegree   int
	// forceFreshStart 标记 restart 上下文（先杀后编译再启动）：端口检查
	// 遇到占用+健康时不得「跳过启动」，必须等待端口释放后启动新二进制
	// （OPT-20260810-005：kill-first 重构，修复 12:15 旧副本残留/宕机缺陷）。
	forceFreshStart bool
	// allowOverlapStart is canary restart (ADR-0058): start a peer on the same
	// port (SO_REUSEPORT) without terminating existing listeners.
	allowOverlapStart   bool
	overlapHadListeners bool
}

type ExecutionLevel struct {
	Services []*ServiceNode
}

func BuildDAG(services []Service) ([]ExecutionLevel, error) {
	if len(services) == 0 {
		return nil, nil
	}

	nodes := make(map[string]*ServiceNode, len(services))
	for _, svc := range services {
		nodes[svc.Name] = &ServiceNode{
			Service:   svc,
			DependsOn: svc.DependsOn,
		}
	}

	// Compute dependents and indegrees
	for _, node := range nodes {
		for _, depName := range node.DependsOn {
			dep, ok := nodes[depName]
			if !ok {
				return nil, fmt.Errorf("service %q depends on unknown service %q", node.Service.Name, depName)
			}
			dep.Dependents = append(dep.Dependents, node.Service.Name)
		}
		node.InDegree = len(node.DependsOn)
	}

	// Kahn's algorithm
	var levels []ExecutionLevel
	processed := 0

	// First level: nodes with indegree 0
	var currentLevel []*ServiceNode
	for _, node := range nodes {
		if node.InDegree == 0 {
			currentLevel = append(currentLevel, node)
		}
	}

	for len(currentLevel) > 0 {
		levels = append(levels, ExecutionLevel{Services: currentLevel})
		processed += len(currentLevel)

		var nextLevel []*ServiceNode
		for _, node := range currentLevel {
			for _, depName := range node.Dependents {
				dep := nodes[depName]
				dep.InDegree--
				if dep.InDegree == 0 {
					nextLevel = append(nextLevel, dep)
				}
			}
		}
		currentLevel = nextLevel
	}

	if processed != len(nodes) {
		for _, node := range nodes {
			if node.InDegree > 0 {
				return nil, fmt.Errorf("cycle detected involving service %q", node.Service.Name)
			}
		}
	}

	return levels, nil
}
