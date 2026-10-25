package main

import (
	"github.com/MustardSeedNetworks/niac-go/internal/protocols/snmp"
	"os"
)

func main() {
 if len(os.Args)==5 && os.Args[1]=="rejections" { if err:=snmp.CorpusRejectionsLocal(os.Args[2],os.Args[3],os.Args[4]);err!=nil{panic(err)};return }
	if len(os.Args) == 4 && os.Args[1] == "identity" {
		if err := snmp.WriteCorpusIdentityRootLocal(os.Args[2], os.Args[3]); err != nil {
			panic(err)
		}
		return
	}
	if len(os.Args) < 2 {
		panic("recording required")
	}
	if len(os.Args) > 2 && os.Args[2] == "net-snmp" {
		snmp.WriteCorpusReplayNetSNMPLocal(os.Args[1])
		return
	}
	snmp.WriteCorpusReplayLocal(os.Args[1], len(os.Args) > 2 && os.Args[2] == "smoke")
}
