package api

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Snoop file format (RFC 1761) is the wireshark-incompatible cousin of
// libpcap. Sun/Solaris and a few older capture tools emit it. NIAC's
// shipped examples/captures/*.pcap files are mislabelled snoop files,
// so the PCAP Inspector previously bailed out on them. This file holds
// a self-contained snoop → pcap converter so the existing pcap-flavour
// analyse + replay code paths can stay untouched.
//
// snoop layout (all multi-byte fields are big-endian):
//
//	file header (16 bytes)
//	  magic            8 bytes  "snoop\0\0\0"
//	  version          4 bytes  current = 2
//	  datalink type    4 bytes  see snoopDLT* below
//
//	per packet record (24-byte header + N bytes data + padding)
//	  original length     4 bytes  size on the wire
//	  included length     4 bytes  bytes captured (== len(data))
//	  packet record len   4 bytes  total bytes of this record (header + data + pad)
//	  cumulative drops    4 bytes  ignored
//	  ts_sec              4 bytes  seconds since epoch
//	  ts_usec             4 bytes  microseconds
//	  data                included_length bytes
//	  pad                 0..3 bytes so record_len is 4-byte aligned

const (
	snoopHeaderLen    = 16
	snoopRecordHeader = 24
	// pcapGlobalHeaderLen and pcapRecordHeaderLen mirror the libpcap
	// on-disk layout that snoopToPCAP emits. Named so the make()
	// allocations below read as intent rather than magic numbers.
	pcapGlobalHeaderLen = 24
	pcapRecordHeaderLen = 16
	// snoopVersionMin / snoopVersionMax: RFC 1761 specifies v2, but
	// newer Solaris / illumos snoop tooling emits v3 / v4 with the
	// same wire layout. We accept the broader range so the existing
	// shipped examples/captures/*.pcap files import cleanly.
	snoopVersionMin = 2
	snoopVersionMax = 4

	// Snoop datalink-type codes (RFC 1761 table) we know how to map.
	snoopDLT802_3    = 0
	snoopDLT802_4    = 1
	snoopDLT802_5    = 2
	snoopDLT802_6    = 3
	snoopDLTEthernet = 4
	snoopDLTHDLC     = 5
	snoopDLTBisync   = 6
	snoopDLTI386     = 7
	snoopDLTFDDI     = 8
	snoopDLTOther    = 9

	// PCAP global-header constants. We emit little-endian classic pcap
	// since that's what most modern tooling (libpcap, tshark, gopacket
	// on x86) reads natively and what NIAC's own pcap path expects.
	pcapWriteVersionMajor uint16 = 2
	pcapWriteVersionMinor uint16 = 4
	pcapWriteSnaplen      uint32 = 262144 // 256 KiB — larger than any sane MTU
)

// snoopMagic is the 8-byte identification pattern at the start of
// every RFC 1761 snoop file: ASCII "snoop" followed by three NULs.
// Stored as a string so it can be a const — there's no Go syntax for
// a const byte slice.
const snoopMagic = "snoop\x00\x00\x00"

var (
	errSnoopShort   = errors.New("snoop file too short for header")
	errSnoopVersion = errors.New("unsupported snoop version")
	errSnoopRecord  = errors.New("snoop record header truncated or invalid")
)

// hasSnoopMagic reports whether the first 8 bytes of data match the
// snoop identification pattern. Cheap probe; callers pre-screen here
// before falling into validatePCAPStructure.
func hasSnoopMagic(data []byte) bool {
	if len(data) < len(snoopMagic) {
		return false
	}
	for i := range len(snoopMagic) {
		if data[i] != snoopMagic[i] {
			return false
		}
	}
	return true
}

// snoopDLTToPCAP maps a snoop datalink-type code onto the equivalent
// libpcap linktype. RFC 1761 enumerates codes 0–9; later Solaris /
// illumos extensions added higher codes (e.g. 10) for IPNET, FibreChannel,
// etc. The vast majority of captures we see in the wild are framed as
// Ethernet regardless of the declared snoop type, so unknown codes
// default to DLT_EN10MB rather than rejecting the file outright. A
// purpose-built snoop reader could be more discriminating; for the
// PCAP Inspector use case, getting the bytes into gopacket beats
// failing closed.
func snoopDLTToPCAP(snoopType uint32) uint32 {
	switch snoopType {
	case snoopDLT802_5:
		return dltIEEE802
	case snoopDLTFDDI:
		return dltFDDI
	default:
		// snoopDLT802_3, snoopDLTEthernet, and any unrecognised code.
		return dltEN10MB
	}
}

// snoopToPCAP converts a snoop-format byte buffer into a classic libpcap
// byte buffer suitable for handing to validatePCAPStructure / gopacket /
// the rest of the analyser. The converter walks every record sequentially
// and re-emits the header + payload in pcap layout; padding bytes are
// dropped, drop counters are dropped, and timestamps are preserved.
//
// Returns the converted bytes plus any error encountered partway through.
// On error we return whatever we've successfully converted so far — the
// caller can decide whether to surface the partial capture or bail.
func snoopToPCAP(data []byte) ([]byte, error) {
	if len(data) < snoopHeaderLen {
		return nil, errSnoopShort
	}
	if !hasSnoopMagic(data) {
		return nil, fmt.Errorf("%w: missing snoop magic", errSnoopShort)
	}

	version := binary.BigEndian.Uint32(data[8:12])
	if version < snoopVersionMin || version > snoopVersionMax {
		return nil, fmt.Errorf("%w: %d (supported range: %d–%d)",
			errSnoopVersion, version, snoopVersionMin, snoopVersionMax)
	}

	snoopDLT := binary.BigEndian.Uint32(data[12:16])
	pcapDLT := snoopDLTToPCAP(snoopDLT)

	// Write classic pcap global header (24 bytes, little-endian).
	// Subtlety: the on-disk magic for a little-endian pcap is the byte
	// sequence [d4 c3 b2 a1], which is what you get when you encode the
	// nominal magic value 0xa1b2c3d4 as little-endian. Encoding the
	// already-byte-swapped constant pcapMagicLittleEndian (0xd4c3b2a1)
	// in LE would double-flip and produce the BE magic, which made the
	// shared validator pick the wrong endianness path.
	out := make([]byte, 0, len(data))
	header := make([]byte, pcapGlobalHeaderLen)
	binary.LittleEndian.PutUint32(header[0:4], pcapMagicBigEndian)
	binary.LittleEndian.PutUint16(header[4:6], pcapWriteVersionMajor)
	binary.LittleEndian.PutUint16(header[6:8], pcapWriteVersionMinor)
	binary.LittleEndian.PutUint32(header[8:12], 0)  // thiszone
	binary.LittleEndian.PutUint32(header[12:16], 0) // sigfigs
	binary.LittleEndian.PutUint32(header[16:20], pcapWriteSnaplen)
	binary.LittleEndian.PutUint32(header[20:24], pcapDLT)
	out = append(out, header...)

	// Walk packet records. The record-length field on each header tells
	// us exactly how far to skip to the next record (data + padding),
	// so we don't need to compute padding ourselves.
	offset := snoopHeaderLen
	for offset < len(data) {
		if offset+snoopRecordHeader > len(data) {
			return out, fmt.Errorf("%w: header at offset %d runs past EOF",
				errSnoopRecord, offset)
		}

		// Indexing into data directly rather than slicing into a
		// recHeader intermediate keeps gosec G602's bounds analysis
		// happy — it can see the per-field offsets against len(data),
		// which the loop guard above already verified.
		origLen := binary.BigEndian.Uint32(data[offset : offset+4])
		inclLen := binary.BigEndian.Uint32(data[offset+4 : offset+8])
		recordLen := binary.BigEndian.Uint32(data[offset+8 : offset+12])
		// data[offset+12:offset+16] is cumulative_drops, unused.
		tsSec := binary.BigEndian.Uint32(data[offset+16 : offset+20])
		tsUsec := binary.BigEndian.Uint32(data[offset+20 : offset+24])

		if recordLen < snoopRecordHeader || int(recordLen) > len(data)-offset {
			return out, fmt.Errorf("%w: bad record length %d at offset %d",
				errSnoopRecord, recordLen, offset)
		}
		if inclLen > origLen {
			return out, fmt.Errorf("%w: included_length %d > original_length %d at offset %d",
				errSnoopRecord, inclLen, origLen, offset)
		}
		if int(snoopRecordHeader+inclLen) > int(recordLen) {
			return out, fmt.Errorf("%w: payload %d would overrun record_len %d at offset %d",
				errSnoopRecord, inclLen, recordLen, offset)
		}

		payloadStart := offset + snoopRecordHeader
		payload := data[payloadStart : payloadStart+int(inclLen)]

		// Emit pcap record: 16-byte header + payload.
		pcapRec := make([]byte, pcapRecordHeaderLen)
		binary.LittleEndian.PutUint32(pcapRec[0:4], tsSec)
		binary.LittleEndian.PutUint32(pcapRec[4:8], tsUsec)
		binary.LittleEndian.PutUint32(pcapRec[8:12], inclLen)
		binary.LittleEndian.PutUint32(pcapRec[12:16], origLen)
		out = append(out, pcapRec...)
		out = append(out, payload...)

		offset += int(recordLen)
	}

	return out, nil
}
