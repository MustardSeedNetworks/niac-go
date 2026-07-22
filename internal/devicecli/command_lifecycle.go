package devicecli

func (s *Session) executeLifecycle(command string) (Response, bool) {
	switch command {
	case "show ip interface brief":
		return Response{Output: renderInterfaces(s.state.Snapshot().Network.Interfaces)}, true
	case "show ip route":
		return Response{Output: renderRoutes(s.state.Snapshot().Network.Routes)}, true
	case "show running-config":
		return Response{Output: renderConfiguration(s.state.Snapshot())}, true
	case "show startup-config":
		return Response{Output: renderConfiguration(s.state.StartupSnapshot())}, true
	case "show configuration events":
		return Response{Output: renderEvents(s.state.Events())}, true
	case "copy running-config startup-config", "write memory":
		s.state.SaveStartup()
		return Response{Output: "Building configuration...\n[OK]"}, true
	case "write erase":
		s.state.EraseStartup()
		return Response{Output: "[OK]"}, true
	case "reload":
		s.state.ReloadStartup()
		return Response{Close: true}, true
	default:
		return Response{}, false
	}
}
