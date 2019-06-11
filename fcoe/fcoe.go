package fcoe

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"

	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
)

const (
	fcoeEtherType = 0x8906
	fipEtherType  = 0x8914
	fcoeMtu       = 9216
)

var (
	allFcfMac    = net.HardwareAddr{0x01, 0x10, 0x18, 0x01, 0x00, 0x02}
	fipTypeNames = []string{"Discovery", "Link Services", "Control", "VLAN", "VN2VN"}
)

type handler interface {
	Handle(sof byte, fc []byte, eof byte)
}

type fcoe struct {
	ifi *net.Interface
	h   handler
}

func New(iface string, h handler) (*fcoe, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, err
	}
	return &fcoe{
		ifi: ifi,
		h:   h,
	}, nil
}

func (f *fcoe) Start() error {
	// TODO(bluecmd): Replace promisc with multicast group joins
	// TODO(bluecmd): Verify MTU is > 2158 or something sane
	c, err := raw.ListenPacket(f.ifi, fipEtherType, nil)
	if err != nil {
		return err
	}
	c.SetPromiscuous(true)
	go f.handleFip(c)

	c, err = raw.ListenPacket(f.ifi, fcoeEtherType, nil)
	if err != nil {
		return err
	}
	c.SetPromiscuous(true)

	go f.handleFcoe(c)
	return nil
}

func (f *fcoe) handleFip(c net.PacketConn) {
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

func (f *fcoe) handleFcoe(c net.PacketConn) {
	var fr ethernet.Frame
	b := make([]byte, fcoeMtu)

	for {
		n, _, err := c.ReadFrom(b)
		if err != nil {
			log.Fatalf("failed to receive message: %v", err)
		}

		// Unpack Ethernet II frame into Go representation.
		if err := (&fr).UnmarshalBinary(b[:n]); err != nil {
			log.Fatalf("failed to unmarshal ethernet frame: %v", err)
		}

		// Display source of message and message itself.
		log.Printf("FCoE [%s -> %s]", fr.Source.String(), fr.Destination.String())
		//fcoeHdr := fr.Payload[0:14]
		sof := fr.Payload[13]
		fc := fr.Payload[14 : len(fr.Payload)-8]
		eof := fr.Payload[len(fr.Payload)-7]
		f.h.Handle(sof, fc, eof)
	}
}
