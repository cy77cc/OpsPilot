package runtime

type ExecutionShape string

const (
	ExecutionShapeSingleAgent         ExecutionShape = "single_agent"
	ExecutionShapeDelegatedSpecialist ExecutionShape = "delegated_specialist"
)

type DispatchDecision struct {
	ExecutionShape ExecutionShape `json:"execution_shape"`
}

type Kernel struct{}

func NewKernel() *Kernel {
	return &Kernel{}
}

func (k *Kernel) DefaultExecutionShape() ExecutionShape {
	return ExecutionShapeSingleAgent
}

func (k *Kernel) BuildDispatchDecision(_ string, specialistAvailable bool) DispatchDecision {
	if specialistAvailable {
		return DispatchDecision{ExecutionShape: ExecutionShapeDelegatedSpecialist}
	}
	return DispatchDecision{ExecutionShape: ExecutionShapeSingleAgent}
}
