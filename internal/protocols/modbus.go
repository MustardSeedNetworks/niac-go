package protocols

import (
	"encoding/binary"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/safeconv"
)

// TCPPortModbus is the registered port for Modbus/TCP.
const TCPPortModbus = 502

// The MBAP header in front of every Modbus/TCP request and response
// (Modbus Messaging on TCP/IP Implementation Guide v1.0b §3.1.3): transaction
// ID, protocol ID, length and unit ID. The length counts the unit ID and the
// PDU, which is at most 253 bytes.
const (
	modbusMBAPLen        = 7
	modbusLengthOffset   = 4
	modbusUnitOffset     = 6
	modbusMinLength      = 2 // unit ID and function code
	modbusMaxLength      = 254
	modbusProtocolModbus = 0
)

// Function codes, exception codes and limits of the Modbus Application
// Protocol Specification v1.1b3 (§6.3, §6.4 and §7).
const (
	modbusReadHoldingRegisters = 0x03
	modbusReadInputRegisters   = 0x04
	modbusExceptionFlag        = 0x80

	modbusIllegalFunction    = 0x01
	modbusIllegalDataAddress = 0x02
	modbusIllegalDataValue   = 0x03

	modbusReadRequestLen  = 5 // function code, starting address, quantity
	modbusReadResponseLen = 2 // function code, byte count
	modbusRegisterLen     = 2
	modbusMaxReadCount    = 125
)

// handleModbus serves 502 on a device with Modbus enabled: register reads are
// how a SCADA master, a historian or an HMI polls a controller. Every other
// device keeps the closed-port RST.
func (h *TCPHandler) handleModbus(
	pkt *Packet,
	ipLayer *layers.IPv4,
	tcp *layers.TCP,
	devices []*config.Device,
) {
	device := h.serviceDevice(devices, ipLayer.DstIP, func(device *config.Device) bool {
		return device.ModbusConfig != nil && device.ModbusConfig.Enabled
	})
	h.serveStream(pkt, ipLayer, tcp, devices, device, h.modbus, func(stream []byte) ([]byte, int, bool) {
		return modbusAnswer(stream, device.ModbusConfig)
	}, "Modbus")
}

// modbusAnswer answers the complete requests at the front of stream. A
// request whose protocol ID is not Modbus is discarded, as the
// implementation guide requires; a length outside what a request can be
// means the stream is not Modbus, and nothing after it is answered.
func modbusAnswer(stream []byte, server *config.ModbusConfig) ([]byte, int, bool) {
	var answer []byte
	consumed := 0
	for len(stream)-consumed >= modbusMBAPLen {
		request := stream[consumed:]
		length := int(binary.BigEndian.Uint16(request[modbusLengthOffset:]))
		if length < modbusMinLength || length > modbusMaxLength {
			return answer, consumed, true
		}
		end := modbusUnitOffset + length
		if len(request) < end {
			break
		}
		consumed += end
		if binary.BigEndian.Uint16(request[2:]) != modbusProtocolModbus {
			continue
		}
		pdu := modbusResponsePDU(request[modbusMBAPLen:end], server)
		header := make([]byte, modbusMBAPLen, modbusMBAPLen+len(pdu))
		copy(header, request[:modbusMBAPLen])
		binary.BigEndian.PutUint16(header[modbusLengthOffset:], safeconv.Uint16(1+len(pdu)))
		answer = append(answer, append(header, pdu...)...)
	}

	return answer, consumed, false
}

// modbusResponsePDU answers one request PDU. Only the two register reads are
// served, over the registers the server has; the checks run in the order of
// the specification's state diagrams, quantity before address.
func modbusResponsePDU(request []byte, server *config.ModbusConfig) []byte {
	function := request[0]
	var registers int
	switch function {
	case modbusReadHoldingRegisters:
		registers = server.HoldingRegisters
	case modbusReadInputRegisters:
		registers = server.InputRegisters
	default:
		return []byte{function | modbusExceptionFlag, modbusIllegalFunction}
	}
	if len(request) != modbusReadRequestLen {
		return []byte{function | modbusExceptionFlag, modbusIllegalDataValue}
	}
	address := int(binary.BigEndian.Uint16(request[1:]))
	quantity := int(binary.BigEndian.Uint16(request[3:]))
	switch {
	case quantity < 1 || quantity > modbusMaxReadCount:
		return []byte{function | modbusExceptionFlag, modbusIllegalDataValue}
	case address+quantity > registers:
		return []byte{function | modbusExceptionFlag, modbusIllegalDataAddress}
	}
	// Every register reads zero, so the values are the zeroed tail.
	values := modbusRegisterLen * quantity
	response := make([]byte, modbusReadResponseLen+values)
	response[0], response[1] = function, safeconv.Byte(values)

	return response
}
