package main

import (
	"fmt"
	"log"
	"os"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	pb "google.golang.org/protobuf/types/descriptorpb"
)

func main() {
	pbBytes, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	fds := &pb.FileDescriptorSet{}
	if err := proto.Unmarshal(pbBytes, fds); err != nil {
		log.Fatal(err)
	}
	m := prototext.MarshalOptions{EmitUnknown: true, Multiline: true}
	// m := protojson.MarshalOptions{Multiline: true}
	b, err := m.Marshal(fds)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
	// opts := fds.File[len(fds.File)-1].Service[0].Method[0].Options
	// fmt.Printf("method options %#v\n", opts)
	// fmt.Println(opts)
	// fmt.Println()
	// rawFields := opts.ProtoReflect().GetUnknown()
	// marshalUnknown(rawFields)
}

// func marshalUnknown(b []byte) {
// 	const dec = 10
// 	const hex = 16
// 	for len(b) > 0 {
// 		num, wtype, n := protowire.ConsumeTag(b)
// 		b = b[n:]
// 		fmt.Println("Name", strconv.FormatInt(int64(num), dec))

// 		switch wtype {
// 		case protowire.VarintType:
// 			var v uint64
// 			v, n = protowire.ConsumeVarint(b)
// 			fmt.Println("Uint", v)
// 		case protowire.Fixed32Type:
// 			var v uint32
// 			v, n = protowire.ConsumeFixed32(b)
// 			fmt.Println("Literal", "0x"+strconv.FormatUint(uint64(v), hex))
// 		case protowire.Fixed64Type:
// 			var v uint64
// 			v, n = protowire.ConsumeFixed64(b)
// 			fmt.Println("Literal", "0x"+strconv.FormatUint(v, hex))
// 		case protowire.BytesType:
// 			var v []byte
// 			v, n = protowire.ConsumeBytes(b)
// 			fmt.Printf("'%s'\n", string(v))
// 		case protowire.StartGroupType:
// 			fmt.Println("StartMessage")
// 			var v []byte
// 			v, n = protowire.ConsumeGroup(num, b)
// 			marshalUnknown(v)
// 			fmt.Println("StartMessage")
// 		default:
// 			panic(fmt.Sprintf("prototext: error parsing unknown field wire type: %v", wtype))
// 		}

// 		b = b[n:]
// 	}
// }
