package model

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestNode(t *testing.T) {
	t.Run("create empty node", func(t *testing.T) {
		node := &Node{
			NodeType: "test",
			NodeID:   1,
		}
		assert.Equal(t, "test", node.NodeType)
		assert.Equal(t, int64(1), node.NodeID)
		assert.Nil(t, node.AddExpr)
		assert.Nil(t, node.AndLogicalExpr)
		assert.Nil(t, node.AssertStmt)
		assert.Nil(t, node.BinaryExpr)
		assert.Nil(t, node.AndBitwiseExpr)
		assert.Nil(t, node.BlockStmt)
		assert.Nil(t, node.BreakStmt)
		assert.Nil(t, node.ClassDecl)
		assert.Nil(t, node.ClassInstanceExpr)
		assert.Nil(t, node.ComparisonExpr)
		assert.Nil(t, node.ContinueStmt)
		assert.Nil(t, node.DivExpr)
		assert.Nil(t, node.DoStmt)
		assert.Nil(t, node.EQExpr)
		assert.Nil(t, node.Field)
		assert.Nil(t, node.FileNode)
		assert.Nil(t, node.ForStmt)
		assert.Nil(t, node.IfStmt)
		assert.Nil(t, node.ImportType)
		assert.Nil(t, node.JavaDoc)
		assert.Nil(t, node.LeftShiftExpr)
		assert.Nil(t, node.MethodDecl)
		assert.Nil(t, node.MethodCall)
		assert.Nil(t, node.MulExpr)
		assert.Nil(t, node.NEExpr)
		assert.Nil(t, node.OrLogicalExpr)
		assert.Nil(t, node.Package)
		assert.Nil(t, node.RightShiftExpr)
		assert.Nil(t, node.RemExpr)
		assert.Nil(t, node.ReturnStmt)
		assert.Nil(t, node.SubExpr)
		assert.Nil(t, node.UnsignedRightShiftExpr)
		assert.Nil(t, node.WhileStmt)
		assert.Nil(t, node.XorBitwiseExpr)
		assert.Nil(t, node.YieldStmt)
	})
}

func TestTreeNode(t *testing.T) {
	t.Run("create empty tree node", func(t *testing.T) {
		node := &TreeNode{
			Node: &Node{
				NodeType: "test",
				NodeID:   1,
			},
		}
		assert.NotNil(t, node.Node)
		assert.Equal(t, "test", node.Node.NodeType)
		assert.Nil(t, node.Parent)
		assert.Empty(t, node.Children)
	})

	t.Run("add single child", func(t *testing.T) {
		parent := &TreeNode{
			Node: &Node{NodeType: "parent", NodeID: 1},
		}
		child := &TreeNode{
			Node: &Node{NodeType: "child", NodeID: 2},
		}

		parent.AddChild(child)

		assert.Len(t, parent.Children, 1)
		assert.Equal(t, child, parent.Children[0])
	})

	t.Run("add multiple children", func(t *testing.T) {
		parent := &TreeNode{
			Node: &Node{NodeType: "parent", NodeID: 1},
		}
		child1 := &TreeNode{
			Node: &Node{NodeType: "child1", NodeID: 2},
		}
		child2 := &TreeNode{
			Node: &Node{NodeType: "child2", NodeID: 3},
		}

		children := []*TreeNode{child1, child2}
		parent.AddChildren(children)

		assert.Len(t, parent.Children, 2)
		assert.Equal(t, child1, parent.Children[0])
		assert.Equal(t, child2, parent.Children[1])
	})

	t.Run("build complex tree structure", func(t *testing.T) {
		root := &TreeNode{
			Node: &Node{NodeType: "root", NodeID: 1},
		}
		
		child1 := &TreeNode{
			Node: &Node{NodeType: "child1", NodeID: 2},
			Parent: root,
		}
		
		child2 := &TreeNode{
			Node: &Node{NodeType: "child2", NodeID: 3},
			Parent: root,
		}
		
		grandchild1 := &TreeNode{
			Node: &Node{NodeType: "grandchild1", NodeID: 4},
			Parent: child1,
		}

		root.AddChild(child1)
		root.AddChild(child2)
		child1.AddChild(grandchild1)

		assert.Len(t, root.Children, 2)
		assert.Len(t, child1.Children, 1)
		assert.Len(t, child2.Children, 0)
		assert.Equal(t, root, child1.Parent)
		assert.Equal(t, root, child2.Parent)
		assert.Equal(t, child1, grandchild1.Parent)
	})
}