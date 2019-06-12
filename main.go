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
	fr.OXID = f.OXID
	fr.RXID = 0xf000
	fr.FCtl = 0x980000
	fr.SeqID = f.SeqID
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
	r.RCtl = 0x23
	r.Type = fc.TypeELS
	r.Destination = fc.Address([3]byte{0xef, 0x01, 0x00})

	r.Payload = []byte{
		0x02, 0x00, 0x00, 0x00, 0x20, 0x06, 0x00, 0x10, 0x13, 0x00,
		0x08, 0x0c, 0x00, 0x00, 0x27, 0x10, 0x00, 0x00, 0x07, 0xd0,
		0x20, 0x0c, 0x00, 0x0d, 0xec, 0x30, 0x98, 0x80, 0x20, 0x01,
		0x00, 0x0d, 0xec, 0x30, 0x98, 0x81, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x88, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x88, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	return p.Send(r)
}
