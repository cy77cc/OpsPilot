package diagnosis

type Report struct {
	Summary         string   `json:"summary"`
	Evidence        []string `json:"evidence,omitempty"`
	RootCauses      []string `json:"root_causes,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
}
