package structs

import "github.com/ngld/fso-table-parser/pkg/parser"

func NewArmorTable() []parser.ContainerItem {
	return []parser.ContainerItem{
		Section("#Armor Type",
			Required(StringValue("$Name")),
			Multi(Section("$Damage Type",
				Required(StringValue("")),
				StringValue("+Calculation"),
				FloatValue("+Value"),
			)),
		),
	}
}
