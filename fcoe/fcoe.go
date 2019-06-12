package fcoe

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	fc "github.com/bluecmd/fibrechannel"
	"github.com/bluecmd/fibrechannel/fcoe"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
)

const (
	fipEtherType   = 0x8914
	fcoeMtu        = 9216
	fcoeMinimalMTU = 2158
)

var (
	ErrTooLowMTU = errors.New("MTU is too low to run FCoE")

	allFcfMac    = net.HardwareAddr{0x01, 0x10, 0x18, 0x01, 0x00, 0x02}
	fipTypeNames = []string{"Discovery", "Link Services", "Control", "VLAN", "VN2VN"}
)

type handler interface {
	Handle(sof fc.SOF, fc []byte, eof fc.EOF)
}

type port struct {
	h    handler
	ifi  *net.Interface
	lock *sync.Mutex
	recv chan *fc.Frame
}

func NewPort(iface string) (*port, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, err
	}
	if ifi.MTU < fcoeMinimalMTU {
		return nil, ErrTooLowMTU
	}
	return &port{
		ifi:  ifi,
		lock: &sync.Mutex{},
		recv: make(chan *fc.Frame, 0),
	}, nil
}

func (p *port) String() string {
	return fmt.Sprintf("FCoE port on %s", p.ifi.Name)
}

func (p *port) Start() error {
	// TODO(bluecmd): Replace promisc with multicast group joins
	// TODO(bluecmd): Verify MTU is > 2158 or something sane
	c, err := raw.ListenPacket(p.ifi, fipEtherType, nil)
	if err != nil {
		return err
	}
	c.SetPromiscuous(true)
	go p.handleFip(c)

	c, err = raw.ListenPacket(p.ifi, fcoe.EtherType, nil)
	if err != nil {
		return err
	}
	c.SetPromiscuous(true)

	go p.handleFcoe(c)
	return nil
}

func (p *port) handleFip(c net.PacketConn) {
	var fr ethernet.Frame
	b := make([]byte, fcoeMtu)

	for {
		n, _, err := c.ReadFrom(b)
		if err != nil {
			log.Fatalf("failed to receive message: %v", err)
		}
		if err := (&fr).UnmarshalBinary(b[:n]); err != nil {
			log.Fatalf("failed to unmarshal ethernet frame: %v", err)
		}

		// TODO(bluecmd): VLAN discovery is not implemented as FCoE in Linux seems to not send it
		log.Printf("FIP [%s -> %s]", fr.Source.String(), fr.Destination.String())
		if bytes.Equal(fr.Destination, allFcfMac) {
			t := binary.BigEndian.Uint16(fr.Payload[2:4])
			log.Printf("FIP Type: %v", fipTypeNames[t-1])
			// TODO(bluecmd):
			// if type == solicitation
			//   send_advertisement(solicitation)
			// if type == flogi
			//   login
			// Linux does not seem to need FIP, so it is not implemented for now
			// Cisco guide-c07-733622.pdf seems to have some good info on how
			// this should work
		}
	}
}

func (p *port) handleFcoe(c net.PacketConn) {
	var fr ethernet.Frame
	var fe fcoe.Frame
	var ff fc.Frame
	b := make([]byte, fcoeMtu)

	for {
		n, _, err := c.ReadFrom(b)
		if err != nil {
			log.Fatalf("failed to receive message: %v", err)
		}

		// Unpack Ethernet II frame
		if err := (&fr).UnmarshalBinary(b[:n]); err != nil {
			log.Printf("failed to unmarshal Ethernet frame: %v", err)
			continue
		}

		// Unpack FCoE frame
		if err := (&fe).UnmarshalBinary(fr.Payload); err != nil {
			log.Printf("failed to unmarshal FCoE frame: %v", err)
			continue
		}

		// Unpack FC frame
		if err := (&ff).UnmarshalBinary(fe.SOF, fe.Payload, fe.EOF); err != nil {
			log.Printf("failed to unmarshal FC frame: %v", err)
			return
		}

		p.recv <- &ff
	}
}

func (p *port) Receive() (*fc.Frame, error) {
	return <-p.recv, nil
}

func (p *port) Send(*fc.Frame) error {
	return errors.New("Send not implemented")
}
