package snmp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// putLength writes a BER-encoded length.
func (w *MibZipWriter) putLength(x int) {
	if x < BERLongFormIndicator {
		w.buf.WriteByte(byte(x))
	} else {
		// Count bytes needed
		size := 1

		tmp := x
		for tmp > BERMaxShortLen+BERLongFormIndicator {
			tmp >>= BitShiftByte
			size++
		}

		w.buf.WriteByte(byte(size | BERHighBitMask))

		for i := size - 1; i >= 0; i-- {
			w.buf.WriteByte(byte((x >> (BitShiftByte * i)) & BERFullByte))
		}
	}
}

// encodeASNValue encodes an ASN.1 value.
func (w *MibZipWriter) encodeASNValue(asnType gosnmp.Asn1BER, value any) {
	w.buf.WriteByte(byte(asnType))

	switch asnType {
	case gosnmp.OctetString:
		w.encodeOctetStringValue(value)
	case gosnmp.Integer:
		w.encodeIntegerValue(value)
	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.Uinteger32, gosnmp.TimeTicks:
		w.encodeUnsigned32Value(value)
	case gosnmp.Counter64:
		w.encodeCounter64Value(value)
	case gosnmp.ObjectIdentifier:
		w.encodeOIDValue(value)
	case gosnmp.IPAddress:
		w.encodeIPAddressValue(value)
	case gosnmp.Null, gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.EndOfMibView:
		w.putLength(0)
	case gosnmp.EndOfContents, // Also covers UnknownType (same value)
		gosnmp.Boolean,
		gosnmp.BitString,
		gosnmp.ObjectDescription,
		gosnmp.Opaque,
		gosnmp.NsapAddress,
		gosnmp.OpaqueFloat,
		gosnmp.OpaqueDouble:
		// Encode as octet string for unsupported types
		str := fmt.Sprintf("%v", value)
		w.putLength(len(str))
		w.buf.WriteString(str)
	}
}

func (w *MibZipWriter) encodeOctetStringValue(value any) {
	str := ""

	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	}

	w.putLength(len(str))
	w.buf.WriteString(str)
}

func (w *MibZipWriter) encodeIntegerValue(value any) {
	var intVal int32

	switch v := value.(type) {
	case int:
		intVal = safeInt32(int64(v))
	case int32:
		intVal = v
	case int64:
		intVal = safeInt32(v)
	}

	w.encodeInteger(int64(intVal))
}

func (w *MibZipWriter) encodeUnsigned32Value(value any) {
	var uintVal uint32

	switch v := value.(type) {
	case uint:
		uintVal = safeUint32FromUint64(uint64(v))
	case uint32:
		uintVal = v
	case uint64:
		uintVal = safeUint32FromUint64(v)
	case int:
		uintVal = safeUint32FromInt(v)
	}

	w.encodeUnsigned(uint64(uintVal))
}

func (w *MibZipWriter) encodeCounter64Value(value any) {
	var uintVal uint64

	switch v := value.(type) {
	case uint64:
		uintVal = v
	case uint:
		uintVal = uint64(v)
	case uint32:
		uintVal = uint64(v)
	}

	w.encodeUnsigned(uintVal)
}

func (w *MibZipWriter) encodeOIDValue(value any) {
	oid := ""

	if v, ok := value.(string); ok {
		oid = v
	}

	encoded := encodeOID(oid)
	w.putLength(len(encoded))
	w.buf.Write(encoded)
}

func (w *MibZipWriter) encodeIPAddressValue(value any) {
	ip := ""

	if v, ok := value.(string); ok {
		ip = v
	}

	parts := strings.Split(ip, ".")

	w.putLength(IPAddressOctets)

	for _, p := range parts {
		n, _ := strconv.Atoi(p)
		w.buf.WriteByte(byte(n))
	}
}

func (w *MibZipWriter) encodeInteger(value int64) {
	// Determine number of bytes needed
	size := 1

	if value < 0 {
		tmp := value
		for tmp < -128 || tmp > BERMaxShortLen {
			tmp >>= 8
			size++
		}
	} else {
		tmp := value
		for tmp > BERMaxShortLen {
			tmp >>= 8
			size++
		}
	}

	w.putLength(size)

	for i := size - 1; i >= 0; i-- {
		w.buf.WriteByte(byte((value >> (BitShiftByte * i)) & BERFullByte))
	}
}

func (w *MibZipWriter) encodeUnsigned(value uint64) {
	size := 1

	tmp := value
	for tmp > 255 {
		tmp >>= 8
		size++
	}
	// Add leading zero if high bit is set (to keep positive)
	if (value >> ((size - 1) * BitShiftByte)) > BERMaxShortLen {
		size++
	}

	w.putLength(size)

	for i := size - 1; i >= 0; i-- {
		w.buf.WriteByte(byte((value >> (BitShiftByte * i)) & BERFullByte))
	}
}
