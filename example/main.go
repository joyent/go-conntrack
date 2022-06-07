package main

import (
	"context"
	"fmt"
	"net"
	"time"

	ct "github.com/florianl/go-conntrack"
	"github.com/mdlayher/netlink"
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

	var filter ct.Con
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

	filter.Origin = &ct.IPTuple{
		Src: &src,
		Dst: &dst,
		Proto: &ct.ProtoTuple{
			Number:  &proto,
			SrcPort: &sp,
			DstPort: &dp,
		},
	}

	filter.Label = &label
	filter.LabelMask = &labelMask

	//fmt.Printf("### Update: %#v\n", filter)

	err = nfct.Update(ct.Conntrack, ct.IPv4, filter)
	if err != nil {
		fmt.Println("could not dump sessions:", err)
		return
	}

	//ExampleGet(&filter)
}

func ExampleGet(filter *ct.Con) {
	nfct, err := ct.Open(&ct.Config{})
	if err != nil {
		fmt.Println("could not create nfct:", err)
		return
	}
	defer nfct.Close()

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
