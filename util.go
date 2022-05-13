package main

import (
	"fmt"
	"strings"

	"github.com/mingmxren/protokit"
)

func Indent(s string, width int) string {
	sb := new(strings.Builder)
	sp := strings.Split(s, "\n")
	for i, s := range sp {
		if len(s) > 0 {
			sb.WriteString(strings.Repeat(" ", width))
		}
		sb.WriteString(s)
		if i != len(sp)-1 {
			sb.WriteString("\n")
		}
	}
	r := sb.String()
	return r
}

func WithComments(s string, comments *protokit.Comment, baseIndex int) string {
	sb := new(strings.Builder)
	for _, c := range comments.Detached {
		sb.WriteString(Indent(fmt.Sprintf("/*\n%s\n  */\n\n", Indent(c, 2)), baseIndex))
	}
	if len(comments.Leading) > 0 {
		sb.WriteString(Indent(fmt.Sprintf("\n\n/*\n%s\n  */\n", Indent(comments.Leading, 2)), baseIndex))
	}
	if len(s) > 0 && s[len(s)-1] == '\n' {
		sb.WriteString(s[:len(s)-1])
	}
	if len(comments.Trailing) > 0 {
		sb.WriteString(fmt.Sprintf(" /* %s */\n", comments.Trailing))
	} else {
		sb.WriteString("\n")
	}
	return sb.String()
}

func GetBuiltinTypeName(field *protokit.PKFieldDescriptor) string {
	builtinTypes := map[int32]string{
		1: "double",
		2: "float",
		3: "int64",
		4: "uint64",
		5: "int32",
		6: "fixed64",
		7: "fixed64",
		8: "bool",
		9: "string",
		// 10: "TYPE_GROUP",
		// 11: "TYPE_MESSAGE",
		12: "bytes",
		13: "uint32",
		// 14: "TYPE_ENUM",
		15: "sfixed32",
		16: "sfixed64",
		17: "sint32",
		18: "sint64",
	}

	if stringType, ok := builtinTypes[int32(field.GetType())]; ok {
		return stringType
	}
	return ""
}

func GetStringType(field *protokit.PKFieldDescriptor) string {
	name := GetBuiltinTypeName(field)
	if name != "" {
		return name
	}

	return field.GetTypeName()
}

func FindMessage(files []*protokit.PKFileDescriptor, name string) *protokit.PKDescriptor {
	for _, f := range files {
		for _, m := range f.GetMessages() {
			if strings.HasPrefix(name, ".") {
				name = name[1:]
			}
			if m.GetFullName() == name {
				return m
			}
		}
	}
	return nil
}

func FindEnum(files []*protokit.PKFileDescriptor, name string) *protokit.PKEnumDescriptor {
	for _, f := range files {
		for _, m := range f.GetEnums() {
			if strings.HasPrefix(name, ".") {
				name = name[1:]
			}
			if m.GetFullName() == name {
				return m
			}
		}
	}
	return nil
}

func LastPart(s string, split string) string {
	i := strings.LastIndex(s, split)
	if i == -1 {
		return s
	}
	return s[i+len(split):]
}
