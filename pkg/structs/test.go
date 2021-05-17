package structs

import "github.com/ngld/fso-table-parser/pkg/parser"

func NewTestTable() []parser.ContainerItem {
	return Join(
		NewArmorTable(),
		NewShipsTable(),
	)
}
