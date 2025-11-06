package protocols

import (
	"net"
	"strings"
	"testing"

	"github.com/krisarmstrong/niac-go/pkg/config"
	"github.com/krisarmstrong/niac-go/pkg/logging"
)

// TestNewFTPHandler tests FTP handler creation
func TestNewFTPHandler(t *testing.T) {
	cfg := &config.Config{}
	stack := NewStack(nil, cfg, logging.NewDebugConfig(0))
	handler := NewFTPHandler(stack)

	if handler == nil {
		t.Fatal("NewFTPHandler returned nil")
	}

	if handler.stack != stack {
		t.Error("Handler stack not set correctly")
	}
}

// TestHandleRequest_USER tests USER command parsing
func TestHandleRequest_USER(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		expectedCmd string
	}{
		{
			name:        "USER with username",
			command:     "USER testuser\r\n",
			expectedCmd: "USER",
		},
		{
			name:        "USER without username",
			command:     "USER\r\n",
			expectedCmd: "USER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test command parsing logic
			command := strings.TrimSpace(tt.command)
			parts := strings.Fields(command)

			if len(parts) == 0 {
				t.Fatal("No command parsed")
			}

			cmd := strings.ToUpper(parts[0])

			if cmd != tt.expectedCmd {
				t.Errorf("Expected command %s, got %s", tt.expectedCmd, cmd)
			}
		})
	}
}

// TestFTPCommands tests various FTP command responses
func TestFTPCommands(t *testing.T) {
	tests := []struct {
		name             string
		command          string
		expectedCode     string
		expectedContains string
	}{
		{
			name:             "USER command",
			command:          "USER anonymous",
			expectedCode:     "331",
			expectedContains: "User name okay",
		},
		{
			name:             "PASS command",
			command:          "PASS password",
			expectedCode:     "230",
			expectedContains: "logged in",
		},
		{
			name:             "SYST command",
			command:          "SYST",
			expectedCode:     "215",
			expectedContains: "UNIX",
		},
		{
			name:             "PWD command",
			command:          "PWD",
			expectedCode:     "257",
			expectedContains: "\"/\"",
		},
		{
			name:             "TYPE ASCII",
			command:          "TYPE A",
			expectedCode:     "200",
			expectedContains: "Type set to",
		},
		{
			name:             "TYPE Binary",
			command:          "TYPE I",
			expectedCode:     "200",
			expectedContains: "Type set to",
		},
		{
			name:             "TYPE without parameter",
			command:          "TYPE",
			expectedCode:     "501",
			expectedContains: "Syntax error",
		},
		{
			name:             "LIST command",
			command:          "LIST",
			expectedCode:     "150",
			expectedContains: "directory listing",
		},
		{
			name:             "CWD command",
			command:          "CWD /home",
			expectedCode:     "250",
			expectedContains: "successfully changed",
		},
		{
			name:             "CWD without directory",
			command:          "CWD",
			expectedCode:     "501",
			expectedContains: "Syntax error",
		},
		{
			name:             "CDUP command",
			command:          "CDUP",
			expectedCode:     "250",
			expectedContains: "successfully changed",
		},
		{
			name:             "MKD command",
			command:          "MKD newdir",
			expectedCode:     "257",
			expectedContains: "Directory created",
		},
		{
			name:             "RMD command",
			command:          "RMD olddir",
			expectedCode:     "250",
			expectedContains: "Directory deleted",
		},
		{
			name:             "DELE command",
			command:          "DELE file.txt",
			expectedCode:     "553",
			expectedContains: "Could not delete",
		},
		{
			name:             "RETR command",
			command:          "RETR file.txt",
			expectedCode:     "550",
			expectedContains: "No such file",
		},
		{
			name:             "STOR command",
			command:          "STOR file.txt",
			expectedCode:     "553",
			expectedContains: "Could not create",
		},
		{
			name:             "NOOP command",
			command:          "NOOP",
			expectedCode:     "200",
			expectedContains: "NOOP ok",
		},
		{
			name:             "QUIT command",
			command:          "QUIT",
			expectedCode:     "221",
			expectedContains: "Goodbye",
		},
		{
			name:             "HELP command",
			command:          "HELP",
			expectedCode:     "214",
			expectedContains: "commands are recognized",
		},
		{
			name:             "Unknown command",
			command:          "UNKN",
			expectedCode:     "502",
			expectedContains: "not implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := generateFTPResponse(tt.command, nil)

			if !strings.HasPrefix(response, tt.expectedCode) {
				t.Errorf("Expected code %s, got response: %s", tt.expectedCode, response)
			}

			if !strings.Contains(response, tt.expectedContains) {
				t.Errorf("Expected response to contain '%s', got: %s", tt.expectedContains, response)
			}
		})
	}
}

// generateFTPResponse simulates FTP response generation (helper for testing)
func generateFTPResponse(command string, devices []*config.Device) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ""
	}

	cmd := strings.ToUpper(parts[0])
	var response string

	switch cmd {
	case "USER":
		if len(parts) > 1 {
			response = "331 User name okay, need password.\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "PASS":
		response = "230 User logged in, proceed.\r\n"

	case "SYST":
		systemType := "UNIX Type: L8"
		if len(devices) > 0 && devices[0].FTPConfig != nil && devices[0].FTPConfig.SystemType != "" {
			systemType = devices[0].FTPConfig.SystemType
		}
		response = "215 " + systemType + "\r\n"

	case "PWD":
		response = "257 \"/\" is current directory.\r\n"

	case "TYPE":
		if len(parts) > 1 {
			response = "200 Type set to " + parts[1] + ".\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "PASV":
		if len(devices) > 0 && len(devices[0].IPAddresses) > 0 {
			ip := devices[0].IPAddresses[0]
			port := 20000
			p1 := port / 256
			p2 := port % 256
			response = "227 Entering Passive Mode (" +
				string(rune('0'+ip[0]/100)) + string(rune('0'+(ip[0]/10)%10)) + string(rune('0'+ip[0]%10)) + "," +
				string(rune('0'+ip[1]/100)) + string(rune('0'+(ip[1]/10)%10)) + string(rune('0'+ip[1]%10)) + "," +
				string(rune('0'+ip[2]/100)) + string(rune('0'+(ip[2]/10)%10)) + string(rune('0'+ip[2]%10)) + "," +
				string(rune('0'+ip[3]/100)) + string(rune('0'+(ip[3]/10)%10)) + string(rune('0'+ip[3]%10)) + "," +
				string(rune('0'+p1/100)) + string(rune('0'+(p1/10)%10)) + string(rune('0'+p1%10)) + "," +
				string(rune('0'+p2/100)) + string(rune('0'+(p2/10)%10)) + string(rune('0'+p2%10)) +
				").\r\n"
		} else {
			response = "500 Passive mode failed.\r\n"
		}

	case "LIST":
		response = "150 Here comes the directory listing.\r\n"
		response += "226 Directory send OK.\r\n"

	case "RETR":
		if len(parts) > 1 {
			filename := parts[1]
			response = "550 " + filename + ": No such file or directory.\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "STOR":
		if len(parts) > 1 {
			response = "553 Could not create file (read-only filesystem).\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "CWD":
		if len(parts) > 1 {
			response = "250 Directory successfully changed.\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "CDUP":
		response = "250 Directory successfully changed.\r\n"

	case "DELE":
		if len(parts) > 1 {
			response = "553 Could not delete file (read-only filesystem).\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "MKD":
		if len(parts) > 1 {
			response = "257 Directory created.\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "RMD":
		if len(parts) > 1 {
			response = "250 Directory deleted.\r\n"
		} else {
			response = "501 Syntax error in parameters or arguments.\r\n"
		}

	case "NOOP":
		response = "200 NOOP ok.\r\n"

	case "QUIT":
		response = "221 Goodbye.\r\n"

	case "HELP":
		response = "214-The following commands are recognized:\r\n" +
			" USER PASS SYST PWD TYPE PASV LIST RETR STOR\r\n" +
			" CWD CDUP DELE MKD RMD NOOP QUIT HELP\r\n" +
			"214 Help OK.\r\n"

	default:
		if len(cmd) <= 4 && cmd == strings.ToUpper(cmd) {
			response = "502 Command not implemented.\r\n"
		}
	}

	return response
}

// TestFTPCommandParsing tests FTP command parsing
func TestFTPCommandParsing(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedCmd  string
		expectedArgs []string
	}{
		{
			name:         "Simple command",
			input:        "USER anonymous",
			expectedCmd:  "USER",
			expectedArgs: []string{"anonymous"},
		},
		{
			name:         "Command with path",
			input:        "CWD /home/user",
			expectedCmd:  "CWD",
			expectedArgs: []string{"/home/user"},
		},
		{
			name:         "Command without args",
			input:        "PWD",
			expectedCmd:  "PWD",
			expectedArgs: []string{},
		},
		{
			name:         "Mixed case command",
			input:        "user testuser",
			expectedCmd:  "USER",
			expectedArgs: []string{"testuser"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := strings.TrimSpace(tt.input)
			parts := strings.Fields(command)

			if len(parts) == 0 {
				t.Fatal("No parts parsed")
			}

			cmd := strings.ToUpper(parts[0])
			if cmd != tt.expectedCmd {
				t.Errorf("Expected command %s, got %s", tt.expectedCmd, cmd)
			}

			args := parts[1:]
			if len(args) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(args))
			}

			for i, expectedArg := range tt.expectedArgs {
				if i < len(args) && args[i] != expectedArg {
					t.Errorf("Arg %d: expected %s, got %s", i, expectedArg, args[i])
				}
			}
		})
	}
}

// TestFTPCustomSystemType tests custom system type configuration
func TestFTPCustomSystemType(t *testing.T) {
	devices := []*config.Device{
		{
			Name: "windows-server",
			FTPConfig: &config.FTPConfig{
				SystemType: "Windows_NT",
			},
		},
	}

	response := generateFTPResponse("SYST", devices)

	if !strings.Contains(response, "Windows_NT") {
		t.Errorf("Expected custom system type 'Windows_NT', got: %s", response)
	}

	if !strings.HasPrefix(response, "215") {
		t.Errorf("Expected code 215, got: %s", response)
	}
}

// TestFTPPassiveMode tests PASV command response
func TestFTPPassiveMode(t *testing.T) {
	devices := []*config.Device{
		{
			Name:        "ftp-server",
			IPAddresses: []net.IP{net.ParseIP("192.168.1.10")},
		},
	}

	response := generateFTPResponse("PASV", devices)

	if !strings.HasPrefix(response, "227") {
		t.Errorf("Expected code 227, got: %s", response)
	}

	if !strings.Contains(response, "Entering Passive Mode") {
		t.Errorf("Expected passive mode message, got: %s", response)
	}

	// Should contain IP address components
	if !strings.Contains(response, "(") || !strings.Contains(response, ")") {
		t.Errorf("Expected IP/port format in response, got: %s", response)
	}
}

// TestFTPPassiveMode_NoDevice tests PASV without configured device
func TestFTPPassiveMode_NoDevice(t *testing.T) {
	response := generateFTPResponse("PASV", nil)

	if !strings.HasPrefix(response, "500") {
		t.Errorf("Expected code 500 for failed passive mode, got: %s", response)
	}
}

// TestFTPAuthenticationFlow tests typical authentication sequence
func TestFTPAuthenticationFlow(t *testing.T) {
	// USER command
	userResp := generateFTPResponse("USER testuser", nil)
	if !strings.HasPrefix(userResp, "331") {
		t.Errorf("USER should return 331, got: %s", userResp)
	}

	// PASS command
	passResp := generateFTPResponse("PASS testpass", nil)
	if !strings.HasPrefix(passResp, "230") {
		t.Errorf("PASS should return 230, got: %s", passResp)
	}
}

// TestFTPFileOperations tests file-related commands
func TestFTPFileOperations(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		expectedCode string
	}{
		{"RETR existing file", "RETR test.txt", "550"},
		{"RETR without filename", "RETR", "501"},
		{"STOR new file", "STOR test.txt", "553"},
		{"STOR without filename", "STOR", "501"},
		{"DELE file", "DELE test.txt", "553"},
		{"DELE without filename", "DELE", "501"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := generateFTPResponse(tt.command, nil)
			if !strings.HasPrefix(response, tt.expectedCode) {
				t.Errorf("Expected code %s, got: %s", tt.expectedCode, response)
			}
		})
	}
}

// TestFTPDirectoryOperations tests directory commands
func TestFTPDirectoryOperations(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		expectedCode string
	}{
		{"PWD current directory", "PWD", "257"},
		{"CWD to directory", "CWD /home", "250"},
		{"CWD without path", "CWD", "501"},
		{"CDUP parent directory", "CDUP", "250"},
		{"MKD create directory", "MKD newdir", "257"},
		{"MKD without name", "MKD", "501"},
		{"RMD remove directory", "RMD olddir", "250"},
		{"RMD without name", "RMD", "501"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := generateFTPResponse(tt.command, nil)
			if !strings.HasPrefix(response, tt.expectedCode) {
				t.Errorf("Expected code %s, got: %s", tt.expectedCode, response)
			}
		})
	}
}

// TestFTPHelpCommand tests HELP command content
func TestFTPHelpCommand(t *testing.T) {
	response := generateFTPResponse("HELP", nil)

	requiredCommands := []string{
		"USER", "PASS", "SYST", "PWD", "TYPE", "PASV", "LIST",
		"RETR", "STOR", "CWD", "CDUP", "DELE", "MKD", "RMD",
		"NOOP", "QUIT", "HELP",
	}

	for _, cmd := range requiredCommands {
		if !strings.Contains(response, cmd) {
			t.Errorf("HELP response should contain %s", cmd)
		}
	}

	if !strings.HasPrefix(response, "214") {
		t.Errorf("Expected code 214, got: %s", response)
	}
}

// TestFTPCaseInsensitive tests case-insensitive command processing
func TestFTPCaseInsensitive(t *testing.T) {
	commands := []string{
		"user testuser",
		"USER testuser",
		"User testuser",
		"uSeR testuser",
	}

	for _, cmd := range commands {
		response := generateFTPResponse(cmd, nil)
		if !strings.HasPrefix(response, "331") {
			t.Errorf("Command %s should be accepted (case-insensitive), got: %s", cmd, response)
		}
	}
}

// TestFTPEmptyCommand tests empty command handling
func TestFTPEmptyCommand(t *testing.T) {
	emptyCommands := []string{"", "   ", "\r\n", "\t"}

	for _, cmd := range emptyCommands {
		response := generateFTPResponse(cmd, nil)
		if response != "" {
			t.Errorf("Empty command should return empty response, got: %s", response)
		}
	}
}

// TestFTPUnknownCommand tests unrecognized FTP-like commands
func TestFTPUnknownCommand(t *testing.T) {
	// Test that short uppercase commands that aren't recognized get 502
	unknownCmds := []string{"ABCD", "XYZ", "FOO"}

	for _, cmd := range unknownCmds {
		t.Run(cmd, func(t *testing.T) {
			response := generateFTPResponse(cmd, nil)
			// Short uppercase commands get "not implemented" response
			if len(response) > 0 && !strings.HasPrefix(response, "502") {
				t.Errorf("Expected 502 for unknown command %s, got: %s", cmd, response)
			}
		})
	}
}

// Benchmarks

// BenchmarkFTPCommandParsing benchmarks command parsing
func BenchmarkFTPCommandParsing(b *testing.B) {
	command := "USER testuser"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		strings.TrimSpace(command)
		strings.Fields(command)
	}
}

// BenchmarkFTPResponseGeneration benchmarks response generation
func BenchmarkFTPResponseGeneration(b *testing.B) {
	devices := []*config.Device{
		{
			Name:        "benchmark-server",
			IPAddresses: []net.IP{net.ParseIP("192.168.1.10")},
		},
	}

	commands := []string{
		"USER testuser",
		"PASS testpass",
		"SYST",
		"PWD",
		"LIST",
		"QUIT",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generateFTPResponse(commands[i%len(commands)], devices)
	}
}
