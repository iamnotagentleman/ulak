package manager

type WorkerStatusResponse struct {
	IsPopulatorActive bool `json:"IsPopulatorActive"`
	IsProcessorActive bool `json:"IsProcessorActive"`
}
