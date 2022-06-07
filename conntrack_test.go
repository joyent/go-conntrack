package conntrack

import (
	"net"
	"testing"

	"github.com/florianl/go-conntrack/internal/unix"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nltest"
)

func TestFlush(t *testing.T) {
	tests := []struct {
		name   string
		family Family
		want   []netlink.Message
	}{
		{name: "Flush IPv4", family: IPv4, want: []netlink.Message{
			{
				Header: netlink.Header{
					Length: 20,
					// NFNL_SUBSYS_CTNETLINK<<8|IPCTNL_MSG_CT_DELETE
					Type: netlink.HeaderType(1<<8 | 2),
					// NLM_F_REQUEST|NLM_F_ACK
					Flags: netlink.Request | netlink.Acknowledge,
					// Can and will be ignored
					Sequence: 0,
					// Can and will be ignored
					PID: nltest.PID,
				},
				// nfgen_family=AF_INET, version=NFNETLINK_V0, res_id=htons(0)
				Data: []byte{0x2, 0x0, 0x0, 0x0},
			},
		},
		},
		{name: "Flush IPv6", family: IPv6, want: []netlink.Message{
			{
				Header: netlink.Header{
					Length: 20,
					// NFNL_SUBSYS_CTNETLINK<<8|IPCTNL_MSG_CT_DELETE
					Type: netlink.HeaderType(1<<8 | 2),
					// NLM_F_REQUEST|NLM_F_ACK
					Flags: netlink.Request | netlink.Acknowledge,
					// Can and will be ignored
					Sequence: 0,
					// Can and will be ignored
					PID: nltest.PID,
				},
				// nfgen_family=AF_INET6, version=NFNETLINK_V0, res_id=htons(0)
				Data: []byte{0xA, 0x0, 0x0, 0x0},
			},
		},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Fake a netfilter conntrack connection
			nfct := &Nfct{}
			AdjustWriteTimeout(nfct, func() error { return nil })
			nfct.Con = nltest.Dial(func(reqs []netlink.Message) ([]netlink.Message, error) {
				if len(reqs) == 0 {
					return nil, nil
				}
				if len(reqs) != 1 {
					t.Fatalf("Expected only one request, got %d", len(reqs))
				}

				// To ignore the Sequence number, we set it to the same value
				tc.want[0].Header.Sequence = reqs[0].Header.Sequence

				if len(reqs) != len(tc.want) {
					t.Fatalf("differen length:\n- want: %#v\n- got: %#v\n", tc.want, reqs)
				}

				for i := 0; i < len(reqs); i++ {
					if len(reqs[i].Data) != len(tc.want[i].Data) {
						t.Fatalf("differen length:\n- want: %#v\n- got: %#v\n", tc.want[i], reqs[i])
					}
					if reqs[i].Header.Type != tc.want[i].Header.Type {
						t.Fatalf("differen header types:\n- want: %#v\n- got: %#v\n", tc.want[i].Header.Type, reqs[i].Header.Type)
					}
					if reqs[i].Header.Flags != tc.want[i].Header.Flags {
						t.Fatalf("differen header flags:\n- want: %#v\n- got: %#v\n", tc.want[i].Header.Flags, reqs[i].Header.Flags)
					}
					for j, v := range reqs[i].Data {
						if v != tc.want[i].Data[j] {
							t.Fatalf("unexpected reply:\n- want: %#v\n-  got: %#v", tc.want[i].Data, reqs[i].Data)
						}
					}
				}
				return nil, nil
			})
			defer nfct.Con.Close()

			if err := nfct.Flush(Conntrack, tc.family); err != nil {
				t.Fatal(err)
			}

		})
	}
}

func TestCreate(t *testing.T) {

	srcIP := net.ParseIP("1.1.1.1")
	dstIP := net.ParseIP("2.2.2.2")
	var tcp uint8 = 17
	var srcPort uint16 = 22
	var dstPort uint16 = 10
	var timeout uint32 = 100
	var tcpState uint8 = 8

	tests := []struct {
		name       string
		attributes Con
		want       []netlink.Message
	}{
		{name: "noAttributes", want: []netlink.Message{
			{
				Header: netlink.Header{
					Length: 20,
					// NFNL_SUBSYS_CTNETLINK<<8|IPCTNL_MSG_CT_NEW
					Type: netlink.HeaderType(1 << 8),
					// NLM_F_REQUEST|NLM_F_CREATE|NLM_F_ACK|NLM_F_EXCL
					Flags: netlink.Request | netlink.Create | netlink.Acknowledge | netlink.Excl,
					// Can and will be ignored
					Sequence: 0,
					// Can and will be ignored
					PID: nltest.PID,
				},
				// nfgen_family=AF_INET, version=NFNETLINK_V0, NFNL_SUBSYS_CTNETLINK
				Data: []byte{0x2, 0x0, 0x0, 0x1},
			},
		}},
		// Example from libnetfilter_conntrack/utils/conntrack_create.c
		{name: "conntrack_create.c", attributes: Con{
			Origin:    &IPTuple{Src: &srcIP, Dst: &dstIP, Proto: &ProtoTuple{Number: &tcp, SrcPort: &srcPort, DstPort: &dstPort}},
			Reply:     &IPTuple{Src: &dstIP, Dst: &srcIP, Proto: &ProtoTuple{Number: &tcp, SrcPort: &dstPort, DstPort: &srcPort}},
			Timeout:   &timeout,
			ProtoInfo: &ProtoInfo{TCP: &TCPInfo{State: &tcpState}}},
			want: []netlink.Message{
				{
					Header: netlink.Header{
						Length: 80,
						// NFNL_SUBSYS_CTNETLINK<<8|IPCTNL_MSG_CT_NEW
						Type: netlink.HeaderType(1 << 8),
						// NLM_F_REQUEST|NLM_F_CREATE|NLM_F_ACK|NLM_F_EXCL
						Flags: netlink.Request | netlink.Create | netlink.Acknowledge | netlink.Excl,
						// Can and will be ignored
						Sequence: 0,
						// Can and will be ignored
						PID: nltest.PID,
					},
					// nfgen_family=AF_INET, version=NFNETLINK_V0, NFNL_SUBSYS_CTNETLINK + netlinkes Attributes
					Data: []byte{0x2, 0x0, 0x0, 0x1, 0x34, 0x0, 0x1, 0x80, 0x14, 0x0, 0x1, 0x80, 0x8, 0x0, 0x1, 0x0, 0x1, 0x1, 0x1, 0x1, 0x8, 0x0, 0x2, 0x0, 0x2, 0x2, 0x2, 0x2, 0x1c, 0x0, 0x2, 0x80, 0x5, 0x0, 0x1, 0x0, 0x11, 0x0, 0x0, 0x0, 0x6, 0x0, 0x2, 0x0, 0x0, 0x16, 0x0, 0x0, 0x6, 0x0, 0x3, 0x0, 0x0, 0xa, 0x0, 0x0, 0x34, 0x0, 0x2, 0x80, 0x14, 0x0, 0x1, 0x80, 0x8, 0x0, 0x1, 0x0, 0x2, 0x2, 0x2, 0x2, 0x8, 0x0, 0x2, 0x0, 0x1, 0x1, 0x1, 0x1, 0x1c, 0x0, 0x2, 0x80, 0x5, 0x0, 0x1, 0x0, 0x11, 0x0, 0x0, 0x0, 0x6, 0x0, 0x2, 0x0, 0x0, 0xa, 0x0, 0x0, 0x6, 0x0, 0x3, 0x0, 0x0, 0x16, 0x0, 0x0, 0x8, 0x0, 0x7, 0x0, 0x0, 0x0, 0x0, 0x64, 0x10, 0x0, 0x4, 0x80, 0xc, 0x0, 0x1, 0x80, 0x5, 0x0, 0x1, 0x0, 0x8, 0x0, 0x0, 0x0},
				},
			}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nfct := &Nfct{}
			AdjustWriteTimeout(nfct, func() error { return nil })
			nfct.Con = nltest.Dial(func(reqs []netlink.Message) ([]netlink.Message, error) {
				if len(reqs) == 0 {
					return nil, nil
				}
				if len(reqs) != 1 {
					t.Fatalf("Expected only one request, got %d", len(reqs))
				}
				// To ignore the Sequence number, we set it to the same value
				tc.want[0].Header.Sequence = reqs[0].Header.Sequence

				if len(reqs) != len(tc.want) {
					t.Fatalf("differen length:\n- want: %#v\n- got: %#v\n", tc.want, reqs)
				}

				for i := 0; i < len(reqs); i++ {
					if len(reqs[i].Data) != len(tc.want[i].Data) {
						t.Fatalf("differen length:\n- want: %#v\n- got: %#v\n", tc.want[i], reqs[i])
					}
					if reqs[i].Header.Type != tc.want[i].Header.Type {
						t.Fatalf("differen header types:\n- want: %#v\n- got: %#v\n", tc.want[i].Header.Type, reqs[i].Header.Type)
					}
					if reqs[i].Header.Flags != tc.want[i].Header.Flags {
						t.Fatalf("differen header flags:\n- want: %#v\n- got: %#v\n", tc.want[i].Header.Flags, reqs[i].Header.Flags)
					}
					for j, v := range reqs[i].Data {
						if v != tc.want[i].Data[j] {
							t.Fatalf("unexpected reply:\n- want: %#v\n-  got: %#v", tc.want[i].Data, reqs[i].Data)
						}
					}
				}
				return nil, nil
			})
			defer nfct.Con.Close()

			if err := nfct.Create(Conntrack, IPv4, tc.attributes); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func testMarshal(t Table, f Family, att Con) error {
	query, err := nestAttributes(nil, &att)
	if err != nil {
		return err
	}
	data := putExtraHeader(uint8(f), unix.NFNETLINK_V0, unix.NFNL_SUBSYS_CTNETLINK)
	data = append(data, query...)

	req := netlink.Message{
		Header: netlink.Header{
			Type:  netlink.HeaderType(t << 8),
			Flags: netlink.Request | netlink.Acknowledge,
		},
		Data: data,
	}

	_ = req

	return nil
}

func BenchmarkMsg(b *testing.B) {
	var filter Con
	src := net.ParseIP("172.30.1.60")
	dst := net.ParseIP("172.30.1.72")
	proto := uint8(6)
	sp := uint16(50965)
	dp := uint16(22)

	label := make([]byte, 16)
	label[0] = 0x11
	label[1] = 0x99
	/*
		label[2] = 22
		label[3] = 33
	*/

	labelMask := make([]byte, 16)
	labelMask[0] = 0xff
	labelMask[1] = 0xff
	/*
		labelMask[2] = 0xff
		labelMask[1] = 0xff
	*/

	filter.Origin = &IPTuple{
		Src: &src,
		Dst: &dst,
		Proto: &ProtoTuple{
			Number:  &proto,
			SrcPort: &sp,
			DstPort: &dp,
		},
	}

	filter.Label = &label
	filter.LabelMask = &labelMask

	for i := 0; i < b.N; i++ {
		testMarshal(Conntrack, IPv4, filter)
	}
}
