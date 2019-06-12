package main

import (
	"log"

	fc "github.com/bluecmd/fibrechannel"
	els "github.com/bluecmd/fibrechannel/els"
	"github.com/bluecmd/fikonfarm/fcoe"
)

const (
	fcTypeEls   = 0x01
	elsCmdFlogi = 0x04
)

type fch struct{}

func main() {
	fh := &fch{}
	// FCoE interface (For now, FCIP coming later)
	f, err := fcoe.New("ens1", fh)
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

func (f *fch) Handle(sof fc.SOF, p []byte, eof fc.EOF) {
	var fr fc.Frame
	if err := (&fr).UnmarshalBinary(sof, p, eof); err != nil {
		log.Printf("failed to unmarshal FC frame: %v", err)
		return
	}
	if fr.Type == fc.TypeELS {
		fe := els.Frame{}
		if err := (&fe).UnmarshalBinary(fr.Payload); err != nil {
			log.Printf("failed to unmarshal ELS frame: %v", err)
			return
		}
		if fe.Command == els.CmdFLOGI {
			if err := f.HandleFLOGI(fe.Payload); err != nil {
				log.Printf("failed to handle FLOGI: %v", err)
				return
			}
		} else {
			log.Printf("Unknown ELS command: 0x%02x", fe.Command)
		}
	} else {
		log.Printf("Unknown FC frame type: 0x%02x", fr.Type)
	}
}

func (f *fch) HandleFLOGI(b []byte) error {
	var fr els.FLOGI
	if err := (&fr).UnmarshalBinary(b); err != nil {
		return err
	}

	log.Printf("FLOGI [%s -> %s]: %+v", fr.WWPN.String(), fr.WWNN.String(), fr)
	return nil
}
