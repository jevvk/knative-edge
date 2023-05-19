package usage

type ClusterUsage struct {
	Cpu    UsageMetric
	Memory UsageMetric

	CpuPressure    Pressure
	MemoryPressure Pressure

	Pods map[string]*PodUsage

	Nodes    map[string]*NodeUsage
	Services map[string]*KServiceUsage
}

func NewClusterUsage() *ClusterUsage {
	return &ClusterUsage{
		Pods:     make(map[string]*PodUsage),
		Nodes:    make(map[string]*NodeUsage),
		Services: make(map[string]*KServiceUsage),
	}
}

func (c *ClusterUsage) FinalizeClusterMetrics() {
	var cpuUsage int64 = 0
	var memoryUsage int64 = 0
	var cpuCapacity int64 = 0
	var memoryCapacity int64 = 0

	for _, node := range c.Nodes {
		cpuUsage += node.Cpu.Usage
		memoryUsage += node.Memory.Usage
		cpuCapacity += node.Cpu.Capacity
		memoryCapacity += node.Memory.Capacity
	}

	c.Cpu.Usage = cpuUsage
	c.Cpu.Capacity = cpuCapacity

	c.Memory.Usage = memoryUsage
	c.Memory.Capacity = memoryCapacity

	if cpuCapacity > 0 {
		c.Cpu.Percentage = float32(cpuUsage) / float32(cpuCapacity) * 100.0
	}

	if memoryCapacity > 0 {
		c.Memory.Percentage = float32(memoryUsage) / float32(memoryCapacity) * 100.0
	}

	nodesWithCpuPressure := 0
	nodesWithMemPressure := 0

	for _, node := range c.Nodes {
		if node.CpuPressure == HighPressure {
			nodesWithCpuPressure++
		}

		if node.MemoryPressure == HighPressure {
			nodesWithMemPressure++
		}
	}

	if float32(nodesWithCpuPressure)/float32(len(c.Nodes))*100 >= NODE_PRESSURE_THRESHOLD {
		c.CpuPressure = HighPressure
	} else {
		c.CpuPressure = LowPressure
	}

	if float32(nodesWithMemPressure)/float32(len(c.Nodes))*100 >= NODE_PRESSURE_THRESHOLD {
		c.MemoryPressure = HighPressure
	} else {
		c.MemoryPressure = LowPressure
	}
}

func (c *ClusterUsage) UpdateFromPreviousState(prev *ClusterUsage) {

}
