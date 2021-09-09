package cmd

type PodInfo struct {
	Name        string
	Namespace   string
	Cluster     string
	Application string
	Uid         string
	CPUMetric   float64
	RAMMetric   float64
	CPULimits   float64
	RAMLimits   float64
	CPURequsts  float64
	RAMRequests float64
	RatingCPU   int
	RatingRAM   int
}

type PodByLimitCPU []PodInfo

type PodByLimitCPUDesc []PodInfo

type PodByLimitRAM []PodInfo

type PodByLimitRAMDesc []PodInfo

type PodByMetricCPU []PodInfo

type PodByMetricCPUDesc []PodInfo

type PodByMetricRAM []PodInfo

type PodByMetricRAMDesc []PodInfo

type PodByRequestsRAM []PodInfo

type PodByRequestsRAMDesc []PodInfo

type PodByRatingCPU []PodInfo

type PodByRatingRAM []PodInfo

type PodByRatingCPUDesc []PodInfo

type PodByRatingRAMDesc []PodInfo

// SetRequestsRating
// Set pod rating from compare requests
func (pod *PodInfo) SetRequestsRating() {
	pod.RatingCPU = 100

	if pod.CPURequsts == 0 {
		pod.RatingCPU = 999
		return
	}
	if pod.CPURequsts < 2*pod.CPUMetric {
		pod.RatingCPU = 5
		return
	}

	if pod.CPURequsts > 3*pod.CPUMetric {
		pod.RatingCPU = 5
	}

}

func (pod *PodInfo) UpdateMetrics(CPU float64, RAM float64) {
	pod.CPUMetric = CPU * 1000
	pod.RAMMetric = RAM / 1024 / 1024
}

// Sorting pods, Limits CPU
func (pods PodByLimitCPU) Len() int { return len(pods) }

func (pods PodByLimitCPU) Less(i, j int) bool {
	return pods[i].CPULimits < pods[j].CPULimits
}

func (pods PodByLimitCPU) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

//

// Sorting pods, Limits RAM
func (pods PodByLimitRAM) Len() int { return len(pods) }

func (pods PodByLimitRAM) Less(i, j int) bool {
	return pods[i].RAMLimits < pods[j].RAMLimits
}

func (pods PodByLimitRAM) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Requests RAM
func (pods PodByRequestsRAM) Len() int { return len(pods) }

func (pods PodByRequestsRAM) Less(i, j int) bool {
	return pods[i].RAMRequests < pods[j].RAMRequests
}

func (pods PodByRequestsRAM) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Requests RAM DESC
func (pods PodByRequestsRAMDesc) Len() int { return len(pods) }

func (pods PodByRequestsRAMDesc) Less(i, j int) bool {
	return pods[i].RAMRequests > pods[j].RAMRequests
}

func (pods PodByRequestsRAMDesc) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Metric CPU
func (pods PodByMetricCPU) Len() int { return len(pods) }

func (pods PodByMetricCPU) Less(i, j int) bool {
	return pods[i].CPUMetric < pods[j].CPUMetric
}

func (pods PodByMetricCPU) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Metric CPU Desc
func (pods PodByMetricCPUDesc) Len() int { return len(pods) }

func (pods PodByMetricCPUDesc) Less(i, j int) bool {
	return pods[i].CPUMetric > pods[j].CPUMetric
}

func (pods PodByMetricCPUDesc) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Limits RAM
func (pods PodByMetricRAM) Len() int { return len(pods) }

func (pods PodByMetricRAM) Less(i, j int) bool {
	return pods[i].RAMMetric < pods[j].RAMMetric
}

func (pods PodByMetricRAM) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Limits RAM DESC
func (pods PodByMetricRAMDesc) Len() int { return len(pods) }

func (pods PodByMetricRAMDesc) Less(i, j int) bool {
	return pods[i].RAMMetric > pods[j].RAMMetric
}

func (pods PodByMetricRAMDesc) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Limits RAM
func (pods PodByRatingRAM) Len() int { return len(pods) }

func (pods PodByRatingRAM) Less(i, j int) bool {
	return pods[i].RatingRAM < pods[j].RatingRAM
}

func (pods PodByRatingRAM) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Limits RAM DESC
func (pods PodByRatingRAMDesc) Len() int { return len(pods) }

func (pods PodByRatingRAMDesc) Less(i, j int) bool {
	return pods[i].RatingRAM > pods[j].RatingRAM
}

func (pods PodByRatingRAMDesc) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Metric CPU
func (pods PodByRatingCPU) Len() int { return len(pods) }

func (pods PodByRatingCPU) Less(i, j int) bool {
	return pods[i].RatingCPU < pods[j].RatingCPU
}

func (pods PodByRatingCPU) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}

// Sorting pods, Metric CPU Desc
func (pods PodByRatingCPUDesc) Len() int { return len(pods) }

func (pods PodByRatingCPUDesc) Less(i, j int) bool {
	return pods[i].RatingCPU > pods[j].RatingCPU
}

func (pods PodByRatingCPUDesc) Swap(i, j int) {
	pods[i], pods[j] = pods[j], pods[i]
}
