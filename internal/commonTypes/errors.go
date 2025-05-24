package commonTypes

type APIError struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}
