package capture

/*
#include <pcap/pcap.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// A compiler's dead handle has no reader to interrupt before closing.
func compileNativeBPFInstructions(linkType layers.LinkType, expr string) ([]pcap.BPFInstruction, error) {
	handle := C.pcap_open_dead(C.int(linkType), C.int(bpfSnaplen))
	if handle == nil {
		return nil, errors.New("open BPF compiler handle")
	}
	defer C.pcap_close(handle)
	filter := C.CString(expr)
	defer C.free(unsafe.Pointer(filter))
	program := (*C.struct_bpf_program)(C.malloc(C.sizeof_struct_bpf_program))
	*program = C.struct_bpf_program{}
	defer C.free(unsafe.Pointer(program))
	defer C.pcap_freecode(program)
	if C.pcap_compile(handle, program, filter, 1, C.PCAP_NETMASK_UNKNOWN) != 0 {
		return nil, fmt.Errorf("compile BPF: %s", C.GoString(C.pcap_geterr(handle)))
	}
	return copyBPFInstructions(program), nil
}

func copyBPFInstructions(program *C.struct_bpf_program) []pcap.BPFInstruction {
	native := unsafe.Slice(program.bf_insns, int(program.bf_len))
	instructions := make([]pcap.BPFInstruction, len(native))
	for i, instruction := range native {
		instructions[i] = pcap.BPFInstruction{
			Code: uint16(instruction.code), Jt: uint8(instruction.jt),
			Jf: uint8(instruction.jf), K: uint32(instruction.k),
		}
	}
	return instructions
}
