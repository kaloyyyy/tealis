package protocol

func SerializeResponse(response string) string {
	return response + "\r\n"
}
