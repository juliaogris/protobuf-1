package compiler

import (
	"fmt"
	"strings"

	"github.com/alecthomas/protobuf/parser"
	pb "google.golang.org/protobuf/types/descriptorpb"
)

// types contains all known proto custom types with their fully
// qualified name (fullname) and serves as a lookup table to get
// fully qualified names for relative names and given scope.
//
//   fullname := .[pkgpart.]*[type.]*[type]
//
// For example:
//
//    package pkg1.pkg2;
//    message Nest {
//      repeated Egg eggs = 1;
//      repeated Egg2 eggs2 = 2;
//      message Egg {
//        optional string chick = 1;
//      }
//    }
//    message Egg2 {
//      optional string duckling = 1;
//    }
//
// The fullname type of eggs and eggs2 would be returned by
//
//     types.fullName("Egg", {"pkg1.pkg2", "Nest"}): .pkg1.pkg2.Nest.Egg
//     types.fullName("Egg2", {"pkg1.pkg2", "Nest"}): .pkg1.pkg2.Egg2
//
// types.fullName returns an empty string if no type can be found.
type optionType int

const (
	none optionType = iota
	file
	message
	field
	oneof
	enum
	enumValue
	service
	method
	extensionRange
)

var stringToOptionType = map[string]optionType{
	"google.protobuf.FileOptions":           file,
	"google.protobuf.MessageOptions":        message,
	"google.protobuf.FieldOptions":          field,
	"google.protobuf.OneofOptions":          oneof,
	"google.protobuf.EnumOptions":           enum,
	"google.protobuf.EnumValueOptions":      enumValue,
	"google.protobuf.ServiceOptions":        service,
	"google.protobuf.MethodOptions":         method,
	"google.protobuf.ExtensionRangeOptions": extensionRange,
}

type types struct {
	// t contains collective types label (MESSAGE, ENUM, GROUP) indexed
	// by fully qualified typename, e.g. pkg.SomeMessage.SomeEnum for
	// later lookup with scope slice, searching from innermost scope
	// outwards: SomeEnum, SomeMessage.SomeEnum,
	// pkg.SomeMessage.SomeEnum, .pkg.SomeMessage.SomeEnum
	t map[string]pb.FieldDescriptorProto_Type
	// save all Extensions of Option Messages e.g.
	// google.protobuf.MethodOptions, google.protobuf.FieldOptions, for
	// later scoped option field lookup.
	intermediateOpts []scopedExtend
	// opts can be looked up by scoped field name, e.g. for
	//   extend google.protobuf.MethodOptions {
	//     HttpRule http = 72295728;
	//   }
	// findOpt(method, "google.api.http") returns
	//  { extNum: 72295728, pbType:MESSAGE, messageType: .google.api.HttpRule }
	//  { extNum: 50000, pbType:MESSAGE, messageType: .pkg.M }
	opts map[optionType]optTable
}

type optField struct {
	extNum      *int32
	pbType      pb.FieldDescriptorProto_Type // MESSAGE, STRING,....
	messageType *string
}

// look up option field extension by full name, .e.g `(google.api.http)`
type optTable map[string]*optField

type scopedExtend struct {
	optType optionType
	scope   []string
	extend  *parser.Extend
}

func newTypes(asts []*ast) *types {
	t := &types{
		t:    map[string]pb.FieldDescriptorProto_Type{},
		opts: map[optionType]optTable{},
	}
	for _, ast := range asts {
		analyseTypes(ast, t)
	}
	t.analyseOpts()
	return t
}

func (t *types) analyseOpts() {
	for _, optExtend := range t.intermediateOpts {
		for _, f := range optExtend.extend.Fields {
			scope, optType := optExtend.scope, optExtend.optType
			pbType, messageTypeStr := fieldType(f, scope, t)
			o := &optField{
				extNum:      fieldTag(f),
				pbType:      pbType,
				messageType: messageTypeStr,
			}
			if t.opts[optType] == nil {
				t.opts[optType] = optTable{}
			}
			name := fieldName(f)
			fName := fullName(name, scope)
			t.opts[optType][fName] = o
		}
	}
}

// func (o *optField) synthesizeMessageDescriptor() *pb.DescriptorProto {
// 	return nil
// }

func (t *types) fullName(typeName string, scope []string) (string, pb.FieldDescriptorProto_Type) {
	if strings.HasPrefix(typeName, ".") {
		if t.t[typeName] != 0 {
			return typeName, t.t[typeName]
		}
		panic(fmt.Sprintf("typeanalysis: not found: %s, %v", typeName, scope))
	}
	for i := len(scope); i >= 0; i-- {
		parts := make([]string, i+1)
		copy(parts, scope[:i])
		parts[i] = typeName
		name := "." + strings.Join(parts, ".")
		if t.t[name] != 0 {
			return name, t.t[name]
		}
	}
	panic(fmt.Sprintf("typeanalysis: not found: %s, %v, %v", typeName, scope, t))
}

func (t *types) addName(relTypeName string, pbType pb.FieldDescriptorProto_Type, scope []string) {
	name := fullName(relTypeName, scope)
	t.t[name] = pbType
}

func fullName(name string, scope []string) string {
	parts := make([]string, len(scope)+1)
	copy(parts, scope)
	parts[len(parts)-1] = name
	return "." + strings.Join(parts, ".")
}

func analyseTypes(ast *ast, t *types) {
	scope := []string{}
	if ast.pkg != "" {
		scope = append(scope, ast.pkg)
	}

	for _, m := range ast.messages {
		analyseMessage(m, scope, t)
	}
	for _, e := range ast.enums {
		t.addName(e.Name, pb.FieldDescriptorProto_TYPE_ENUM, scope)
	}
	for _, e := range ast.extends {
		analyseExtend(e, scope, t)
	}
}

func analyseMessage(m *parser.Message, scope []string, t *types) {
	name := m.Name
	t.addName(name, pb.FieldDescriptorProto_TYPE_MESSAGE, scope)
	scope = append(scope, name)
	analyseMessageEntries(m.Entries, scope, t)
}

func analyseGroup(g *parser.Group, scope []string, t *types) {
	name := g.Name
	t.addName(name, pb.FieldDescriptorProto_TYPE_GROUP, scope)
	scope = append(scope, name)
	analyseMessageEntries(g.Entries, scope, t)
}

func analyseExtend(e *parser.Extend, scope []string, t *types) {
	if optType := stringToOptionType[e.Reference]; optType != none {
		scopedExt := scopedExtend{extend: e, scope: scope, optType: optType}
		t.intermediateOpts = append(t.intermediateOpts, scopedExt)
	}
	for _, f := range e.Fields {
		if f.Group != nil {
			analyseGroup(f.Group, scope, t)
		}
	}
}

func analyseField(f *parser.Field, scope []string, t *types) {
	if f.Group != nil {
		analyseGroup(f.Group, scope, t)
		return
	}
	if f.Direct != nil && f.Direct.Type != nil && f.Direct.Type.Map != nil {
		mapType := mapTypeStr(f.Direct.Name)
		t.addName(mapType, pb.FieldDescriptorProto_TYPE_MESSAGE, scope)
	}
}

func analyseMessageEntries(messageEntries []*parser.MessageEntry, scope []string, t *types) {
	for _, me := range messageEntries {
		switch {
		case me.Message != nil:
			analyseMessage(me.Message, scope, t)
		case me.Enum != nil:
			t.addName(me.Enum.Name, pb.FieldDescriptorProto_TYPE_ENUM, scope)
		case me.Extend != nil:
			analyseExtend(me.Extend, scope, t)
		case me.Field != nil:
			analyseField(me.Field, scope, t)
		}
	}
}
