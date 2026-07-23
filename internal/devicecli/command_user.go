package devicecli

func (s *Session) executeUser(fields []string) Response {
	if len(fields) != 1 {
		return Response{Output: invalidInput}
	}
	switch fields[0] {
	case "enable":
		s.mode = ModePrivileged
		return Response{}
	case commandExit, "logout":
		return Response{Close: true}
	default:
		return Response{Output: invalidInput}
	}
}
