package model

type BaseTop interface {
	GetAPrimaryQlClass() string
	GetFile() string
	GetLocation() Location
	GetNumberOfCommentLines() int
	GetNumberOfLinesOfCode() int
	GetPrimaryQlClasses() string
	GetTotalNumberOfLines() int
	HasLocationInfo(filepath string, startline, startcolumn, endline, endcolumn int) bool
	ToString() string
}

type Top struct {
	BaseTop
	File string
}

type ControlFlowNode struct {
	Top
}

type BasicBlock struct {
	ControlFlowNode
}
