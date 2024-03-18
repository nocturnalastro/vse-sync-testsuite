package clients

type TargetType int

const (
	TargetUnknown TargetType = iota - 1
	TargetOCP
	TargetLocal
)

var runTimeTarget TargetType = TargetUnknown

func SetRuntimeTarget(target TargetType) {
	runTimeTarget = target
}

func GetRuntimeTarget() TargetType {
	return runTimeTarget
}
