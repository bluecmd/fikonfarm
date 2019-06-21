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

	// TODO(bluecmd): This is configurable to allow for running multiple
	// SANs on the same L2. If somebody ever needs that functionality,
	// it can be added.
	fcMap = [3]byte{0x0e, 0xfc, 0x00}

	allFcfMac    = net.HardwareAddr{0x01, 0x10, 0x18, 0x01, 0x00, 0x02}
	fipTypeNames = []string{"Discovery", "Link Services", "Control", "VLAN", "VN2VN"}
)

type packetSource interface {
	ReadFrom([]byte) (int, net.Addr, error)
}

type packetSink interface {
	WriteTo([]byte, net.Addr) (int, error)
}

type handler interface {
	Handle(sof fc.SOF, fc []byte, eof fc.EOF)
}

type port struct {
	h    handler
	lock *sync.RWMutex
	recv chan *fc.Frame
	// These are identical and only needed for write, but
	// there is no real interface to create a generic send
	// socket so we have two.
	fcoe packetSink
	fip  packetSink
	peer net.HardwareAddr
	name string
}

type realPort struct {
	ifi *net.Interface
	port
}

type fakePort struct {
	port
}

func NewPort(iface string) (*realPort, error) {
	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, err
	}
	if ifi.MTU < fcoeMinimalMTU {
		return nil, ErrTooLowMTU
	}
	p := &realPort{
		ifi: ifi,
		port: port{
			lock: &sync.RWMutex{},
			recv: make(chan *fc.Frame, 0),
			name: fmt.Sprintf("FCoE/%s", ifi.Name),
		},
	}

	if err := p.start(); err != nil {
		return nil, err
	}

	return p, nil
}

func NewFakePort() *fakePort {
	return &fakePort{
		port: port{
			lock: &sync.RWMutex{},
			recv: make(chan *fc.Frame, 0),
			name: fmt.Sprintf("FCoE/Fake"),
		},
	}
}

func (p *fakePort) Put([]byte) {

}

func (p *port) String() string {
	return p.name
}

func (p *realPort) start() error {
	var err error
	// TODO(bluecmd): Replace promisc with multicast group joins
	// TODO(bluecmd): Verify MTU is > 2158 or something sane
	fip, err := raw.ListenPacket(p.ifi, fipEtherType, nil)
	if err != nil {
		return err
	}
	fip.SetPromiscuous(true)
	p.fip = fip
	go p.handleFip(fip)

	fcoe, err := raw.ListenPacket(p.ifi, fcoe.EtherType, nil)
	if err != nil {
		return err
	}
	fcoe.SetPromiscuous(true)
	p.fcoe = fcoe
	go p.handleFcoe(fcoe)
	return nil
}

func (p *port) handleFip(c packetSource) {
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

		if !p.validatePeer(&fr.Source) {
			continue
		}

		// TODO(bluecmd): VLAN discovery is not implemented as FCoE in Linux seems to not send it
		log.Printf("[%s] FIP [%s -> %s]", p.String(), fr.Source.String(), fr.Destination.String())
		if bytes.Equal(fr.Destination, allFcfMac) {
			t := binary.BigEndian.Uint16(fr.Payload[2:4])
			log.Printf("[%s] FIP Type: %v", p.String(), fipTypeNames[t-1])
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

// FC is using point-to-point link, and thus FCoE is that as well.
// Given that Ethernet has multiple talkers it is possible we have multiple
// FCoE hosts on the same L2 due to misconfiguration. Lock to processing from
// a single peer and warn if multiple ones are detected.
func (p *port) validatePeer(pr *net.HardwareAddr) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.peer) == 0 {
		p.peer = make([]byte, 6)
		copy(p.peer[:], *pr)
		log.Printf("[%s] Learned peer %s", p.String(), pr.String())
	} else if !bytes.Equal(p.peer, *pr) {
		log.Printf("[%s] WARNING: Ignoring other peer %s", p.String(), pr.String())
		return false
	}
	return true
}

func (p *port) handleFcoe(c packetSource) {
	var fr ethernet.Frame
	var fe fcoe.Frame
	var ff fc.Frame
	b := make([]byte, fcoeMtu)

	for {
		n, _, err := c.ReadFrom(b)
		if err != nil {
			log.Fatalf("[%s] failed to receive message: %v", p.String(), err)
		}

		// Unpack Ethernet II frame
		if err := (&fr).UnmarshalBinary(b[:n]); err != nil {
			log.Printf("[%s] failed to unmarshal Ethernet frame: %v", p.String(), err)
			continue
		}

		if !p.validatePeer(&fr.Source) {
			continue
		}

		// Unpack FCoE frame
		if err := (&fe).UnmarshalBinary(fr.Payload); err != nil {
			log.Printf("[%s] failed to unmarshal FCoE frame: %v", p.String(), err)
			continue
		}

		// Unpack FC frame
		if err := (&ff).UnmarshalBinary(fe.SOF, fe.Payload, fe.EOF); err != nil {
			log.Printf("[%s] failed to unmarshal FC frame: %v", p.String(), err)
			return
		}

		p.recv <- &ff
	}
}

func (p *port) sendFcoe(f *fc.Frame) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	var src [6]byte
	copy(src[:], fcMap[:])
	copy(src[3:], f.Source[:])

	fb, err := f.MarshalBinary()
	if err != nil {
		return err
	}

	fe := fcoe.Frame{
		Version: 0,
		SOF:     f.SOF,
		EOF:     f.EOF,
		Payload: fb,
	}
	fe.CRC32 = fe.Checksum()
	feb, err := fe.MarshalBinary()
	if err != nil {
		return err
	}

	ef := &ethernet.Frame{
		Destination: p.peer,
		Source:      src[:],
		EtherType:   ethernet.EtherType(fcoe.EtherType),
		Payload:     feb,
	}

	b, err := ef.MarshalBinary()
	if err != nil {
		return err
	}

	// Lag one frame to update the destination MAC
	// This means that the flow of FLOGI -> ACC will result in that the p.peer
	// being set to what the peer address is if we have one.
	if !bytes.Equal([]byte{0, 0, 0}, f.Destination[:]) {
		copy(p.peer[:], fcMap[:])
		copy(p.peer[3:], f.Destination[:])
		log.Printf("[%s] Switched peer to %s", p.String(), p.peer.String())
	}

	// Required by Linux, even though the Ethernet frame has a destination.
	// Unused by BSD.
	addr := &raw.Addr{
		HardwareAddr: ef.Destination,
	}
	_, err = p.fcoe.WriteTo(b, addr)
	return err
}

func (p *port) Receive() (*fc.Frame, error) {
	return <-p.recv, nil
}

func (p *port) Send(f *fc.Frame) error {
	return p.sendFcoe(f)
}
