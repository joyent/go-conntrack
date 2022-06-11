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
	//ExampleUpdate()
	//ExampleNfct_Dump()

	GetConnByMark()
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

func dumpCon(con *ct.Con) {
	org := con.Origin

	src := org.Src.String()
	dst := org.Dst.String()
	proto := org.Proto

	var sp, dp uint16
	var zone uint16
	var mark uint32

	if con.Zone != nil {
		zone = *con.Zone
	}

	if con.Mark != nil {
		mark = *con.Mark
	}

	if proto.SrcPort != nil {
		sp = *proto.SrcPort
	}

	if proto.DstPort != nil {
		dp = *proto.DstPort
	}

	var label []byte
	if con.Label != nil {
		label = *con.Label
	}

	fmt.Printf(">>> con:%+v, org:%+v, reply: %+v \n", con, con.Origin, con.Reply)

	if *proto.Number == 1 {
		var id uint16
		var t, c uint8

		if proto.IcmpID != nil {
			id = *proto.IcmpID
		}

		if proto.IcmpType != nil {
			t = *proto.IcmpType
		}

		if proto.IcmpCode != nil {
			c = *proto.IcmpCode
		}

		fmt.Printf(">>> %s => %s, id=%d, type=%d, code=%d, zone=%d, mark=%d, label=%v \n",
			src, dst, id, t, c, zone, mark, label)

	} else {
		fmt.Printf(">>> %s:%d => %s:%d, zone=%d, mark=%d, label=%v \n", src, sp, dst, dp, zone, mark, label)
	}
}

func updateCon(nfct *ct.Nfct, con *ct.Con) {

	fmt.Printf("### Update con \n")
	timestamp := uint32(time.Now().Unix())

	label := make([]byte, 16)
	binary.LittleEndian.PutUint32(label[1:5], timestamp)
	label[0] = 33

	labelMask := make([]byte, 16)
	binary.LittleEndian.PutUint32(labelMask[1:5], ^uint32(0))
	labelMask[0] = 0xff

	con.Label = &label
	con.LabelMask = &labelMask

	con.Reply = nil
	con.Mark = nil

	err := nfct.Update(ct.Conntrack, ct.IPv4, *con)
	if err != nil {
		fmt.Println("### error UpdateBatch:", err)
	}
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

		if (*session.Origin.Proto.Number == 6 &&
			*session.Origin.Proto.DstPort == 22) ||
			session.Zone != nil {
			dumpCon(&session)
			updateCon(nfct, &session)
		}

		if session.Label != nil {
			//dumpCon(&session)
			//fmt.Printf("%#v\n", session)
			//fmt.Printf("### Label: %+v \n", session.Label)
		}
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
			fmt.Println("could not Get sessions:", err)
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

func ExampleQuery(filter ct.FilterAttr) {
	nfct, err := ct.Open(&ct.Config{})
	if err != nil {
		fmt.Println("could not create nfct:", err)
		return
	}
	defer nfct.Close()

	sessions, err := nfct.Query(ct.Conntrack, ct.IPv4, filter)
	if err != nil {
		fmt.Println("could not Query sessions:", err)
		return
	}

	for i, session := range sessions {
		fmt.Printf("### Query: %d\n", i)
		dumpCon(&session)

		if session.Label != nil {
			fmt.Printf("### Label: %+v \n", session.Label)
		}
	}
}

func GetConnByMark() {
	var q ct.FilterAttr
	q.Mark = make([]byte, 4)
	q.MarkMask = make([]byte, 4)
	q.MarkMask[0] = 0xff
	q.MarkMask[1] = 0xff
	q.MarkMask[2] = 0xff
	q.MarkMask[3] = 0xff

	binary.BigEndian.PutUint32(q.Mark, 4)

	ExampleQuery(q)
}
