package core

const (
	PacketTypeData byte = iota
	PacketTypeIPAssign
)

type IPAssignBody struct {
	IP string `json:"ip"`
}
