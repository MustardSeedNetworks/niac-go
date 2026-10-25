package snmp

import (
	"bufio"
	"compress/bzip2"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"github.com/gosnmp/gosnmp"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func WriteCorpusIdentityRootLocal(root, output string) error {
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err = writer.Write([]string{"path", "source_sha256", "sysdescr_type", "sysdescr_nonempty", "sysobjectid_type", "sysobjectid_valid", "typed_identity"}); err != nil {
		return err
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "repaired/") && !strings.HasPrefix(rel, "repaired/ready/") && !strings.HasPrefix(rel, "repaired/numeric/") {
			return nil
		}
		if !strings.HasSuffix(path, ".walk") && !strings.HasSuffix(path, ".snmprec") && !strings.HasSuffix(path, ".snmprec.bz2") {
			return nil
		}
		bytes, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		digest := fmt.Sprintf("%x", sha256.Sum256(bytes))
		bytes = nil
		descr, object, e := corpusIdentityFile(path)
		if e != nil {
			return e
		}
		dt, ot := gosnmp.Asn1BER(0), gosnmp.Asn1BER(0)
		hasDesc, hasObject := false, false
		if descr != nil {
			dt = descr.Type
			b, ok := octets(descr.Value)
			hasDesc = ok && len(b) > 0 && dt == gosnmp.OctetString
		}
		if object != nil {
			ot = object.Type
			hasObject = ot == gosnmp.ObjectIdentifier && validWireOID(fmt.Sprint(object.Value))
		}
		return writer.Write([]string{rel, digest, strconv.Itoa(int(dt)), strconv.FormatBool(hasDesc), strconv.Itoa(int(ot)), strconv.FormatBool(hasObject), strconv.FormatBool(hasDesc && hasObject)})
	})
	if err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func corpusIdentityFile(path string) (*WalkEntry, *WalkEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	var reader io.Reader = f
	if strings.HasSuffix(path, ".bz2") {
		reader = bzip2.NewReader(f)
	}
	buffered := bufio.NewReader(reader)
	recording := contentIsSnmprec(buffered)
	scanner := bufio.NewScanner(buffered)
	scanner.Buffer(make([]byte, 65536), 4<<20)
	var descr, object *WalkEntry
	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || isEndOfMIBLine(line) {
			continue
		}
		var oid string
		if recording {
			oid, _, _ = strings.Cut(raw, "|")
		} else {
			oid, _, _ = strings.Cut(line, "=")
		}
		oid = strings.TrimPrefix(strings.TrimSpace(oid), ".")
		if strings.Contains(oid, "::") || (len(oid) > 0 && (oid[0] < '0' || oid[0] > '9')) {
			if normalized, ok := NormalizeWalkOID(oid); ok {
				oid = strings.TrimPrefix(normalized, ".")
			}
		}
		if oid != "1.3.6.1.2.1.1.1.0" && oid != "1.3.6.1.2.1.1.2.0" {
			continue
		}
		var entry *WalkEntry
		if recording {
			entry, err = parseSnmprecLine(raw)
		} else {
			entry, err = parseWalkLine(line)
		}
		if err != nil {
			continue
		}
		if oid == "1.3.6.1.2.1.1.1.0" {
			descr = entry
		} else {
			object = entry
		}
	}
	return descr, object, scanner.Err()
}
