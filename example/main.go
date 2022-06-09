package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	ct "github.com/florianl/go-conntrack"
	"github.com/mdlayher/netlink"
	"github.com/openlyinc/pointy"
)

func main() {
	ExampleUpdate()
}

func ExampleEvent() {
	nfct, err := ct.Open(&ct.Config{})
	if err != nil {
		fmt.Println("could not create nfct:", err)
		return
	}
	defer nfct.Close()

	nfct.Con.SetOption(netlink.ListenAllNSID, true)
	nfct.Con.SetOption(netlink.NoENOBUFS, true)

	var cnt int
	monitor := func(c ct.Con) int {

		cnt++
		if cnt%10000 == 0 {
			fmt.Printf("%d: %#v\n", cnt, c)
		}

		return 0
	}

	//if err := nfct.Register(context.Background(), ct.Expected, ct.NetlinkCtExpectedNew|ct.NetlinkCtExpectedUpdate|ct.NetlinkCtExpectedDestroy, monitor); err != nil {
	err = nfct.Register(context.Background(),
		ct.Conntrack,
		ct.NetlinkCtNew|ct.NetlinkCtUpdate|ct.NetlinkCtDestroy,
		monitor)

	if err != nil {
		fmt.Println("could not register callback:", err)
		return
	}

	time.Sleep(3600 * time.Second)

}

func ExampleNfct_Dump() {
	nfct, err := ct.Open(&ct.Config{})
	if err != nil {
		fmt.Println("could not create nfct:", err)
		return
	}
	defer nfct.Close()

	sessions, err := nfct.Dump(ct.Conntrack, ct.IPv4)
	if err != nil {
		fmt.Println("could not dump sessions:", err)
		return
	}

	for _, session := range sessions {
		fmt.Printf("%#v\n", session)
	}
}

func ExampleUpdate() {
	nfct, err := ct.Open(&ct.Config{})
	if err != nil {
		fmt.Println("could not create nfct:", err)
		return
	}
	defer nfct.Close()

	var filter []*ct.Con

	timestamp := uint32(time.Now().Unix())

	if true {
		f1 := ct.Con{}
		src := net.ParseIP("172.30.1.60")
		dst := net.ParseIP("172.30.1.72")
		proto := uint8(6)
		sp := uint16(51137)
		dp := uint16(22)

		label := make([]byte, 16)
		binary.LittleEndian.PutUint32(label[0:4], timestamp)
		//label[0] = 11
		//label[1] = 22

		labelMask := make([]byte, 16)
		binary.LittleEndian.PutUint32(labelMask[0:4], ^uint32(0))
		//labelMask[0] = 0xff
		//labelMask[1] = 0xff

		f1.Origin = &ct.IPTuple{
			Src: &src,
			Dst: &dst,
			Proto: &ct.ProtoTuple{
				Number:  &proto,
				SrcPort: &sp,
				DstPort: &dp,
			},
		}

		f1.Label = &label
		f1.LabelMask = &labelMask

		filter = append(filter, &f1)
	}

	if false {
		f2 := ct.Con{}
		src := net.ParseIP("172.30.1.60")
		dst := net.ParseIP("172.30.1.72")
		proto := uint8(6)
		sp := uint16(53044)
		dp := uint16(22)

		label := make([]byte, 16)
		binary.LittleEndian.PutUint32(label[1:5], timestamp)
		label[0] = 33
		//label[1] = 22

		labelMask := make([]byte, 16)
		binary.LittleEndian.PutUint32(labelMask[1:5], ^uint32(0))
		labelMask[0] = 0xff
		//labelMask[1] = 0xff

		f2.Origin = &ct.IPTuple{
			Src: &src,
			Dst: &dst,
			Proto: &ct.ProtoTuple{
				Number:  &proto,
				SrcPort: &sp,
				DstPort: &dp,
			},
		}

		f2.Label = &label
		f2.LabelMask = &labelMask

		filter = append(filter, &f2)
	}

	//fmt.Printf("### Update: %#v\n", filter)

	cnt := 100

	//////////////////////////

	for i := 1; i < cnt; i++ {
		f := filter[0]
		n := *f

		n.Origin.Proto.SrcPort = pointy.Uint16(*n.Origin.Proto.SrcPort + 1)

		filter = append(filter, &n)
	}

	start := time.Now()
	err = nfct.UpdateBatch(ct.Conntrack, ct.IPv4, filter)
	if err != nil {
		fmt.Println("error UpdateBatch:", err)
		return
	}
	elapsed := time.Since(start)
	fmt.Printf("### UpdateBatch(%d attrs) took %s \n", len(filter), elapsed)

	///////////////////////////////////////

	start = time.Now()
	err = nfct.UpdateSingle(ct.Conntrack, ct.IPv4, filter)
	if err != nil {
		fmt.Println("error UpdateSingle:", err)
		return
	}
	elapsed = time.Since(start)
	fmt.Printf("### UpdateSingle(%d attrs) took %s \n", len(filter), elapsed)

	////////////////

	//ExampleGet(filter)
}

func ExampleGet(filters []*ct.Con) {
	nfct, err := ct.Open(&ct.Config{})
	if err != nil {
		fmt.Println("could not create nfct:", err)
		return
	}
	defer nfct.Close()

	for _, filter := range filters {
		sessions, err := nfct.Get(ct.Conntrack, ct.IPv4, *filter)
		if err != nil {
			fmt.Println("could not dump sessions:", err)
			return
		}

		for i, session := range sessions {
			fmt.Printf("### %d: %#v\n", i, session)
			if session.Label != nil {
				fmt.Printf("### Label: %+v \n", session.Label)
			}
		}
	}
}
