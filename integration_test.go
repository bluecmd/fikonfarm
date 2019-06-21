package main

import (
	"io"
	"testing"

	"github.com/bluecmd/fikonfarm/fcoe"
	"github.com/google/gopacket/pcap"
)

func TestFCOE(t *testing.T) {
	sw := NewSwitch()

	handle, err := pcap.OpenOffline("test/fcoe1-init.pcap")
	if err != nil {
		t.Fatalf("Failed to open test pcap: %v", err)
	}

	f := fcoe.NewFakePort()
	sw.AddPort(f, PortTypeF)

	for {
		data, _, err := handle.ReadPacketData()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed while processing pcap: %v", err)
		}
		// TODO(bluecmd): Implement put to actually do something
		f.Put(data)
		// TODO(bluecmd): Compare the output on F with fcoe1-fabric.pcap
	}
}
