package gorm

type ApaasModeType uint8

const (
	DirectMode ApaasModeType = iota
	EngineMode
)

func (m ApaasModeType) String() string {
	switch m {
	case DirectMode:
		return "DirectMode"
	case EngineMode:
		return "EngineMode"
	}
	return "DirectMode"
}
