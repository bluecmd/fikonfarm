package main

import (
	"log"
	"sync"

	fc "github.com/bluecmd/fibrechannel"
	els "github.com/bluecmd/fibrechannel/els"
	"github.com/bluecmd/fikonfarm/fcoe"
)

const (
	PortTypeF PortType = 1
	PortTypeE PortType = 2
)

type PortType int

type Port interface {
	Send(*fc.Frame) error
	Receive() (*fc.Frame, error)
	String() string
}

type sanSwitch struct {
	lock *sync.Mutex
	p    []Port
	pt   []PortType
}

func main() {
	sw := NewSwitch()

	// Add one FCoE F_port
	f, err := fcoe.NewPort("ens1")
	if err != nil {
		log.Fatalf("FCoE port creation failed: %v", err)
	}
	if err := f.Start(); err != nil {
		log.Fatalf("FCoE port starting failed: %v", err)
	}

	sw.AddPort(f, PortTypeF)

	// Block forever.
	select {}
}

func NewSwitch() *sanSwitch {
	sw := &sanSwitch{
		lock: &sync.Mutex{},
	}
	return sw
}

func (sw *sanSwitch) AddPort(p Port, pt PortType) {
	sw.lock.Lock()
	defer sw.lock.Unlock()
	sw.p = append(sw.p, p)
	sw.pt = append(sw.pt, pt)
	go sw.portRecv(p, pt)
	log.Printf("Added port %s to switch", p)
}

func (sw *sanSwitch) portRecv(p Port, pt PortType) {
	for {
		f, err := p.Receive()
		if err != nil {
			log.Printf("Port %s failed: %v", p.String(), err)
			return
		}

		if f.Type == fc.TypeELS {
			fe := els.Frame{}
			if err := (&fe).UnmarshalBinary(f.Payload); err != nil {
				log.Printf("failed to unmarshal ELS frame: %v", err)
				return
			}
			if fe.Command == els.CmdFLOGI {
				if err := sw.handleFLOGI(p, f, fe.Payload); err != nil {
					log.Printf("failed to handle FLOGI: %v", err)
					return
				}
			} else {
				log.Printf("Unknown ELS command: 0x%02x", fe.Command)
			}
		} else {
			log.Printf("Unknown FC frame type: 0x%02x", f.Type)
		}
	}
}

func (sw *sanSwitch) newReply(f *fc.Frame) *fc.Frame {
	var fr fc.Frame
	// TODO(bluecmd): Check that the SOF/EOF
	fr.SOF = f.SOF
	fr.EOF = f.EOF

	fr.CSCtl = new(fc.ClassControl)
	fr.Source = f.Destination
	fr.Destination = f.Source
	// TODO more
	return &fr
}

func (sw *sanSwitch) handleFLOGI(p Port, f *fc.Frame, b []byte) error {
	var fr els.FLOGI
	if err := (&fr).UnmarshalBinary(b); err != nil {
		return err
	}

	log.Printf("FLOGI [%s -> %s]: %+v", fr.WWPN.String(), fr.WWNN.String(), fr)
	//f.p.Send(...)
	// 1. Assign FCID
	// 2. Buffer credits?
	// 3. Send ACC

	r := sw.newReply(f)
	r.Destination = fc.Address([3]byte{0xef, 0x01, 0x00})
	return p.Send(r)
}
