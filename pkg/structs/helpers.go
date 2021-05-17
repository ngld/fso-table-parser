package structs

import (
	"github.com/ngld/fso-table-parser/pkg/parser"
)

func Section(name string, properties ...parser.ContainerChild) parser.ContainerItem {
	return parser.ContainerItem{
		Name:       name,
		Multi:      false,
		Value:      nil,
		Properties: properties,
	}
}

func BooleanSection(name string, properties ...parser.ContainerChild) parser.ContainerItem {
	return parser.ContainerItem{
		Name:             name,
		Multi:            false,
		Value:            nil,
		BooleanContainer: true,
		Properties:       properties,
	}
}

func StringValue(name string) parser.ContainerItem {
	return parser.ContainerItem{
		Name:  name,
		Value: parser.StringValue,
	}
}

func WordValue(name string) parser.ContainerItem {
	return parser.ContainerItem{
		Name:  name,
		Value: parser.WordValue,
	}
}

func StringListValue(name string) parser.ContainerItem {
	return parser.ContainerItem{
		Name: name,
		Value: parser.ValueList{
			ValueParser: parser.StringFlag,
		},
	}
}

func StringFlagsValue(name string, flags ...string) parser.ContainerItem {
	// TODO: Validate flags
	return parser.ContainerItem{
		Name: name,
		Value: parser.ValueList{
			ValueParser: parser.StringValue,
		},
	}
}

func MultilineStringValue(name string) parser.ContainerItem {
	return parser.ContainerItem{
		Name:  name,
		Value: parser.MultilineStringValue,
	}
}

func IntegerValue(name string) parser.ContainerItem {
	return parser.ContainerItem{
		Name:  name,
		Value: parser.IntegerValue,
	}
}

func FloatValue(name string) parser.ContainerItem {
	return parser.ContainerItem{
		Name:  name,
		Value: parser.FloatValue,
	}
}

func FloatListValue(name string, count int) parser.ContainerItem {
	return parser.ContainerItem{
		Name: name,
		Value: parser.ValueList{
			ValueParser: parser.FloatValue,
		},
	}
}

func Vec3dValue(name string) parser.ContainerItem {
	return parser.ContainerItem{
		Name:  name,
		Value: parser.Vec3dValue,
	}
}

func ColorValue(name string) parser.ContainerItem {
	return parser.ContainerItem{
		Name:  name,
		Value: parser.ColorValue,
	}
}

func IntegerListValue(name string, count int) parser.ContainerItem {
	return parser.ContainerItem{
		Name: name,
		Value: parser.ValueList{
			ValueParser: parser.IntegerValue,
		},
	}
}

func BooleanFlag(name string) parser.ContainerItem {
	return parser.ContainerItem{
		Name:  name,
		Value: parser.FlagValue,
	}
}

func BooleanValue(name string) parser.ContainerItem {
	return parser.ContainerItem{
		Name:  name,
		Value: parser.BooleanValue,
	}
}

func BooleanListValue(name string) parser.ContainerItem {
	return parser.ContainerItem{
		Name: name,
		Value: parser.ValueList{
			ValueParser: parser.BooleanValue,
		},
	}
}

func EnumValue(name string, values ...string) parser.ContainerItem {
	// TODO: validate
	return parser.ContainerItem{
		Name: name,
		Value: parser.ValueList{
			ValueParser: parser.StringValue,
		},
	}
}

func VoidValue(name string) parser.ContainerItem {
	return parser.ContainerItem{Name: name}
}

func Either(items ...parser.ContainerChild) parser.ContainerChild {
	return &parser.SwitchItem{Items: items}
}

func Required(item parser.ContainerItem) parser.ContainerItem {
	item.Required = true
	return item
}

func Multi(item parser.ContainerItem) parser.ContainerItem {
	item.Multi = true
	return item
}

func Deprecated(item parser.ContainerItem, msg string) parser.ContainerItem {
	item.DeprecatedMessage = msg
	return item
}

func Nocreate() parser.ContainerItem {
	return BooleanFlag("+nocreate")
}

func Join(containers ...[]parser.ContainerItem) []parser.ContainerItem {
	result := make([]parser.ContainerItem, 0)
	for _, part := range containers {
		result = append(result, part...)
	}

	return result
}
