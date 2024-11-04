package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTop(t *testing.T) {
	top := &Top{
		File: "test.go",
	}

	assert.Equal(t, "test.go", top.File)
}

func TestControlFlowNode(t *testing.T) {
	node := &ControlFlowNode{
		Top: Top{
			File: "test.go",
		},
	}

	assert.Equal(t, "test.go", node.File)
}

func TestBasicBlock(t *testing.T) {
	block := &BasicBlock{
		ControlFlowNode: ControlFlowNode{
			Top: Top{
				File: "test.go",
			},
		},
	}

	assert.Equal(t, "test.go", block.File)
}

type MockTop struct {
	BaseTop
	file                  string
	location              Location
	numberOfCommentLines  int
	numberOfLinesOfCode   int
	primaryQlClasses      string
	totalNumberOfLines    int
	hasLocationInfoResult bool
}

func (m *MockTop) GetAPrimaryQlClass() string {
	return "MockClass"
}

func (m *MockTop) GetFile() string {
	return m.file
}

func (m *MockTop) GetLocation() Location {
	return m.location
}

func (m *MockTop) GetNumberOfCommentLines() int {
	return m.numberOfCommentLines
}

func (m *MockTop) GetNumberOfLinesOfCode() int {
	return m.numberOfLinesOfCode
}

func (m *MockTop) GetPrimaryQlClasses() string {
	return m.primaryQlClasses
}

func (m *MockTop) GetTotalNumberOfLines() int {
	return m.totalNumberOfLines
}

func (m *MockTop) HasLocationInfo(filepath string, startline, startcolumn, endline, endcolumn int) bool {
	return m.hasLocationInfoResult
}

func (m *MockTop) ToString() string {
	return "MockTop"
}

func TestBaseTopInterface(t *testing.T) {
	mock := &MockTop{
		file:                  "test.go",
		numberOfCommentLines:  10,
		numberOfLinesOfCode:   50,
		primaryQlClasses:      "TestClass",
		totalNumberOfLines:    60,
		hasLocationInfoResult: true,
	}

	t.Run("GetAPrimaryQlClass", func(t *testing.T) {
		assert.Equal(t, "MockClass", mock.GetAPrimaryQlClass())
	})

	t.Run("GetFile", func(t *testing.T) {
		assert.Equal(t, "test.go", mock.GetFile())
	})

	t.Run("GetNumberOfCommentLines", func(t *testing.T) {
		assert.Equal(t, 10, mock.GetNumberOfCommentLines())
	})

	t.Run("GetNumberOfLinesOfCode", func(t *testing.T) {
		assert.Equal(t, 50, mock.GetNumberOfLinesOfCode())
	})

	t.Run("GetPrimaryQlClasses", func(t *testing.T) {
		assert.Equal(t, "TestClass", mock.GetPrimaryQlClasses())
	})

	t.Run("GetTotalNumberOfLines", func(t *testing.T) {
		assert.Equal(t, 60, mock.GetTotalNumberOfLines())
	})

	t.Run("HasLocationInfo", func(t *testing.T) {
		assert.True(t, mock.HasLocationInfo("test.go", 1, 1, 10, 10))
	})

	t.Run("ToString", func(t *testing.T) {
		assert.Equal(t, "MockTop", mock.ToString())
	})
}
