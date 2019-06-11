package main

import (
	"log"

	"github.com/bluecmd/fikonfarm/fcoe"
)

const (
	fcTypeEls   = 0x01
	elsCmdFlogi = 0x04
)

type fc struct{}

func main() {
	fc := &fc{}
	// FCoE interface (For now, FCIP coming later)
	f, err := fcoe.New("ens1", fc)
	if err != nil {
		log.Fatalf("FCoE creation failed: %v", err)
	}

	err = f.Start()
	if err != nil {
		log.Fatalf("FCoE start failed: %v", err)
	}

	// Block forever.
	select {}
}

func (f *fc) Handle(sof byte, fc []byte, eof byte) {
	t := fc[8]
	if t == fcTypeEls {
		els := fc[24:]
		cmd := els[0]
		if cmd == elsCmdFlogi {
			log.Printf("FLOGI: %v", els)
			// TODO(bluecmd): Implement login
		} else {
			log.Printf("Unknown ELS command: 0x%02x", cmd)
		}
	} else {
		log.Printf("Unknown FC frame type: 0x%02x", t)
	}
}
