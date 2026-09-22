package bridge

// Request struct
type Request struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Wait    bool     `json:"wait"`
}

// Response struct
type Response struct {
	OK       bool   `json:"ok"`
	PID      int    `json:"pid"`
	ExitCode int    `json:"exitCode"`
	Message  string `json:"message"`
}

// NewRequest creates a new request object
func NewRequest() *Request {
	return &Request{
		Command: "",
		Args:    make([]string, 0),
		Wait:    false,
	}
}

// NewResponse creates a new response object
func NewResponse() *Response {
	return &Response{
		OK:       false,
		PID:      0,
		ExitCode: 1,
		Message:  "",
	}
}
