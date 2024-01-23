package v1beta1

const (
	ConditionTypeError = "error"
	ConditionTypeReady = "ready"

	ReasonInvalidKymaName = "InvalidKymaName"
	ReasonUnknown         = "Unknown"
	ReasonReady           = "Ready"
	ReasonGcpError        = "GCPError"
	ReasonNotSupported    = "NotSupported"
)