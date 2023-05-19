package usage

type Pressure int

const (
	LowPressure Pressure = iota
	HighPressure
)

func (p Pressure) String() string {
	switch p {
	case LowPressure:
		return "LowPressure"
	case HighPressure:
		return "HighPressure"
	default:
		return "null"
	}
}

const (
	CPU_PRESSURE_THRESHOLD  float32 = 80.0
	MEM_PRESSURE_THRESHOLD  float32 = 80.0
	NODE_PRESSURE_THRESHOLD float32 = 66
)
