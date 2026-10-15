package protocols

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

const (
	ftpSyntaxErrorResponse = "501 Syntax error in parameters or arguments.\r\n"
	ftpHelpResponse        = "214-The following commands are recognized:\r\n" +
		" USER PASS SYST PWD TYPE PASV EPSV LIST RETR STOR\r\n" +
		" CWD CDUP DELE MKD RMD NOOP QUIT HELP\r\n214 Help OK.\r\n"

	// ftpMaxCommandLine bounds what a client can make the server hold while
	// it waits for the end of a command line.
	ftpMaxCommandLine = 4096

	// ftpPassivePort is the data port PASV and EPSV name.
	ftpPassivePort      = 20000
	ftpPortByteMultiply = 256
)

// handleFTP serves TCP 21 on a device with FTP enabled. Every other device
// keeps the closed-port RST. FTP speaks first: the greeting goes out on the
// ACK that completes the handshake, and every command line after it is
// answered in order.
func (h *TCPHandler) handleFTP(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
) {
	device := h.serviceDevice(devices, ipLayer.DstIP, servesFTP)
	if device != nil && tcp.ACK && !tcp.SYN && !tcp.FIN && !tcp.RST && len(tcp.Payload) == 0 {
		if h.ftp.greet(newStreamKey(ipLayer, tcp)) {
			h.sendStatelessSegment(
				pkt,
				ipLayer,
				tcp,
				device,
				[]byte(ftpGreeting(device.FTPConfig.WelcomeBanner)),
				"FTP",
			)
		}
		return
	}
	address := ipLayer.DstIP.To4()
	h.serveStream(pkt, ipLayer, tcp, devices, device, h.ftp, func(stream []byte) ([]byte, int, bool) {
		return ftpAnswer(stream, device.FTPConfig, address)
	}, "FTP")
}

func servesFTP(device *config.Device) bool {
	return device.FTPConfig != nil && device.FTPConfig.Enabled
}

// ftpGreeting is the authored 220 reply, ended with the CRLF every FTP reply
// line carries.
func ftpGreeting(banner string) string {
	if strings.HasSuffix(banner, "\r\n") {
		return banner
	}

	return banner + "\r\n"
}

// ftpAnswer answers the complete command lines at the front of stream. QUIT
// ends the conversation, and so does a line longer than ftpMaxCommandLine.
func ftpAnswer(stream []byte, cfg *config.FTPConfig, address net.IP) ([]byte, int, bool) {
	var answer []byte
	consumed := 0
	for {
		end := bytes.IndexByte(stream[consumed:], '\n')
		if end < 0 {
			if len(stream)-consumed > ftpMaxCommandLine {
				return append(answer, "500 Command line too long.\r\n"...), len(stream), true
			}
			return answer, consumed, false
		}
		line := strings.TrimSpace(string(stream[consumed : consumed+end]))
		consumed += end + 1
		if line == "" {
			continue
		}
		command, arg, _ := strings.Cut(line, " ")
		command = strings.ToUpper(command)
		answer = append(answer, ftpReply(command, strings.TrimSpace(arg), cfg, address)...)
		if command == "QUIT" {
			return answer, consumed, true
		}
	}
}

// ftpReply is the reply to one command. Nothing is stored and no data
// connection is served.
func ftpReply(command, arg string, cfg *config.FTPConfig, address net.IP) string {
	switch command {
	case "USER":
		return handleFTPUser(arg)
	case "PASS":
		return "230 User logged in, proceed.\r\n"
	case "SYST":
		return fmt.Sprintf("215 %s\r\n", cfg.SystemType)
	case "PWD":
		return "257 \"/\" is current directory.\r\n"
	case "TYPE":
		return handleFTPType(arg)
	case "PASV":
		return handleFTPPasv(address)
	case "EPSV":
		return fmt.Sprintf("229 Entering Extended Passive Mode (|||%d|).\r\n", ftpPassivePort)
	case "LIST":
		return "150 Here comes the directory listing.\r\n226 Directory send OK.\r\n"
	case "RETR":
		return handleFTPRetr(arg)
	case "STOR":
		return handleFTPStor(arg)
	case "CWD":
		return handleFTPCwd(arg)
	case "CDUP":
		return "250 Directory successfully changed.\r\n"
	case "DELE":
		return handleFTPDele(arg)
	case "MKD":
		return handleFTPMkd(arg)
	case "RMD":
		return handleFTPRmd(arg)
	case "NOOP":
		return "200 NOOP ok.\r\n"
	case "QUIT":
		return "221 Goodbye.\r\n"
	case "HELP":
		return ftpHelpResponse
	default:
		return "502 Command not implemented.\r\n"
	}
}

func handleFTPUser(arg string) string {
	if arg != "" {
		return "331 User name okay, need password.\r\n"
	}

	return ftpSyntaxErrorResponse
}

func handleFTPType(arg string) string {
	if arg != "" {
		return fmt.Sprintf("200 Type set to %s.\r\n", arg)
	}

	return ftpSyntaxErrorResponse
}

func handleFTPPasv(address net.IP) string {
	return fmt.Sprintf("227 Entering Passive Mode (%d,%d,%d,%d,%d,%d).\r\n",
		address[0], address[1], address[2], address[3],
		ftpPassivePort/ftpPortByteMultiply, ftpPassivePort%ftpPortByteMultiply)
}

func handleFTPRetr(arg string) string {
	if arg != "" {
		return fmt.Sprintf("550 %s: No such file or directory.\r\n", arg)
	}

	return ftpSyntaxErrorResponse
}

func handleFTPStor(arg string) string {
	if arg != "" {
		return "553 Could not create file (read-only filesystem).\r\n"
	}

	return ftpSyntaxErrorResponse
}

func handleFTPCwd(arg string) string {
	if arg != "" {
		return "250 Directory successfully changed.\r\n"
	}

	return ftpSyntaxErrorResponse
}

func handleFTPDele(arg string) string {
	if arg != "" {
		return "553 Could not delete file (read-only filesystem).\r\n"
	}

	return ftpSyntaxErrorResponse
}

func handleFTPMkd(arg string) string {
	if arg != "" {
		return "257 Directory created.\r\n"
	}

	return ftpSyntaxErrorResponse
}

func handleFTPRmd(arg string) string {
	if arg != "" {
		return "250 Directory deleted.\r\n"
	}

	return ftpSyntaxErrorResponse
}
