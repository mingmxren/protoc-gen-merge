package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/mingmxren/protokit"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
	"gopkg.in/yaml.v2"
)

type PluginOptions struct {
	MainProto         string   `yaml:"main_proto"`
	AdditionalMessage []string `yaml:"additional_message"`
	AdditionalEnum    []string `yaml:"additional_enum"`
}

func (po *PluginOptions) ParseOptions(parameter string) {
	yamlFile, err := ioutil.ReadFile(parameter)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(yamlFile, po)
	if err != nil {
		log.Fatal(err)
	}
}

func NewPluginOptions() *PluginOptions {
	po := new(PluginOptions)
	return po
}

type Plugin struct {
	Opts *PluginOptions
}

func NewPlugin() (pi *Plugin) {
	pi = new(Plugin)
	pi.Opts = NewPluginOptions()

	return pi
}

func (pi *Plugin) Generate(req *pluginpb.CodeGeneratorRequest) (*pluginpb.CodeGeneratorResponse, error) {
	rsp := &pluginpb.CodeGeneratorResponse{
		SupportedFeatures: proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)),
	}
	// only one parameter: a yaml file name
	pi.Opts.ParseOptions(req.GetParameter())

	allFiles, err := protokit.ParseCodeGenRequestAllFiles(req)
	if err != nil {
		return nil, err
	}
	for _, pf := range allFiles {
		if pf.GetName() == pi.Opts.MainProto {
			if rf, err := pi.merge(pf, allFiles); err != nil {
				return nil, err
			} else {
				rsp.File = append(rsp.File, rf)
			}
		}
	}

	return rsp, nil
}

func (pi *Plugin) merge(pf *protokit.PKFileDescriptor,
	files []*protokit.PKFileDescriptor) (*pluginpb.CodeGeneratorResponse_File, error) {
	rf := &pluginpb.CodeGeneratorResponse_File{}

	rf.Name = proto.String(LastPart(pf.GetName(), "/"))

	pb := &strings.Builder{}

	pb.WriteString(WithComments(fmt.Sprintf("syntax = \"proto3\";\n"), pf.SyntaxComments, 0))
	pb.WriteString(WithComments(fmt.Sprintf("package %s;\n", pf.GetPackage()), pf.PackageComments, 0))

	if pf.GetOptions().GetCcGenericServices() {
		pb.WriteString("option cc_generic_services = true;\n")
	}

	for _, service := range pf.GetServices() {
		pb.WriteString(WithComments(pi.GenServiceDefine(service), service.Comments, 0))
	}
	enums := make([]*protokit.PKEnumDescriptor, 0)
	enums = append(enums, pf.GetEnums()...)
	msgs := make([]*protokit.PKDescriptor, 0)
	msgs = append(msgs, pf.GetMessages()...)

	for _, am := range pi.Opts.AdditionalMessage {
		m := FindMessage(files, am)
		if m != nil {
			msgs = append(msgs)
		} else {
			return nil, fmt.Errorf("message %s not found", am)
		}
	}

	for _, am := range pi.Opts.AdditionalEnum {
		m := FindEnum(files, am)
		if m != nil {
			enums = append(enums)
		} else {
			return nil, fmt.Errorf("enum %s not found", am)
		}
	}
	for {
		tmpMsgs := make([]*protokit.PKDescriptor, 0)
		for _, m := range msgs {
			for _, f := range m.GetMessageFields() {
				if f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
					m := FindMessage(files, f.GetTypeName())
					if m != nil {
						found := false
						for _, mm := range msgs {
							if mm == m {
								found = true
							}
						}
						if !found {
							tmpMsgs = append(tmpMsgs, m)
						}
					} else {
						return nil, fmt.Errorf("message %s not found", f.GetTypeName())
					}
				} else if f.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM {
					e := FindEnum(files, f.GetTypeName())
					if e != nil {
						found := false
						for _, ee := range enums {
							if ee == e {
								found = true
							}
						}
						if !found {
							enums = append(enums, e)
						}
					} else {
						return nil, fmt.Errorf("enum %s not found", f.GetTypeName())
					}
				}
			}
		}
		msgs = append(msgs, tmpMsgs...)
		if len(tmpMsgs) == 0 {
			break
		}
	}

	for _, enum := range enums {
		pb.WriteString(WithComments(pi.GenEnumDefine(enum), enum.Comments, 0))
	}

	for _, msg := range msgs {
		pb.WriteString(WithComments(pi.GenMessageDefine(msg), msg.Comments, 0))
	}

	rf.Content = proto.String(pb.String())

	return rf, nil
}

func (pi *Plugin) GenServiceDefine(service *protokit.PKServiceDescriptor) string {
	sb := new(strings.Builder)
	sb.WriteString(fmt.Sprintf("service %s {\n", service.GetName()))
	for _, method := range service.GetMethods() {
		sb.WriteString(WithComments(Indent(pi.GenMethodDefine(method), 4), method.Comments, 4))
	}

	sb.WriteString("}\n")

	return sb.String()

}

func (pi *Plugin) GenMessageDefine(msg *protokit.PKDescriptor) string {
	sb := new(strings.Builder)
	sb.WriteString(fmt.Sprintf("message %s {\n", msg.GetName()))
	for _, subMsg := range msg.GetMessages() {
		sb.WriteString(Indent(WithComments(pi.GenMessageDefine(subMsg), subMsg.Comments, 4), 4))
	}
	for _, subEnum := range msg.GetEnums() {
		sb.WriteString(Indent(WithComments(pi.GenEnumDefine(subEnum), subEnum.Comments, 4), 4))
	}
	for _, field := range msg.GetMessageFields() {
		sb.WriteString(WithComments(fmt.Sprintf("    %s %s %s = %d;\n", pi.GetStringLabel(field.GetLabel()),
			pi.ReplacePackage(GetStringType(field)), field.GetName(), field.GetNumber()), field.Comments, 4))
	}
	sb.WriteString("}\n")
	return sb.String()
}
func (pi *Plugin) GenEnumDefine(enum *protokit.PKEnumDescriptor) string {
	sb := new(strings.Builder)
	sb.WriteString(fmt.Sprintf("enum %s {\n", enum.GetName()))
	for _, val := range enum.GetValues() {
		sb.WriteString(WithComments(fmt.Sprintf("    %s = %d;\n", val.GetName(), val.GetNumber()), val.Comments, 4))
	}
	sb.WriteString("}\n")
	return sb.String()
}

func (pi *Plugin) GenMethodDefine(method *protokit.PKMethodDescriptor) string {
	sb := new(strings.Builder)
	sb.WriteString(fmt.Sprintf("rpc %s(%s) returns (%s) {\n", method.GetName(),
		pi.ReplacePackage(method.GetInputType()), pi.ReplacePackage(method.GetOutputType())))
	for optName, opt := range method.OptionExtensions {
		sb.WriteString(fmt.Sprintf("    option (%s) = \"%s\";\n", optName, opt))
	}
	sb.WriteString("}\n")

	return sb.String()
}

func (pi *Plugin) GetStringLabel(label descriptorpb.FieldDescriptorProto_Label) string {
	if label == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		return "repeated"
	}

	if label == descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL {
		return ""
	}
	return ""
}

func (pi *Plugin) ReplacePackage(name string) string {
	return LastPart(name, ".")
}