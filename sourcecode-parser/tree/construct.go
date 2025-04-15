package graph

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/db"
	javalang "github.com/shivasurya/code-pathfinder/sourcecode-parser/tree/java"
	utilities "github.com/shivasurya/code-pathfinder/sourcecode-parser/util"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"github.com/smacker/go-tree-sitter/java"

	sitter "github.com/smacker/go-tree-sitter"
)

func buildQLTreeFromAST(node *sitter.Node, sourceCode []byte, file string, parentNode *model.TreeNode, storageNode *db.StorageNode) {
	switch node.Type() {
	case "import_declaration":
		importDeclNode := javalang.ParseImportDeclaration(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{ImportType: importDeclNode, NodeType: "ImportType", NodeID: 1}, Parent: parentNode})
		storageNode.AddImportDecl(importDeclNode)
	case "package_declaration":
		packageDeclNode := javalang.ParsePackageDeclaration(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{Package: packageDeclNode, NodeType: "Package", NodeID: 2}, Parent: parentNode})
		storageNode.AddPackage(packageDeclNode)
	case "block":
		blockStmtNode := javalang.ParseBlockStatement(node, sourceCode)
		blockStmtTreeNode := &model.TreeNode{Node: &model.Node{BlockStmt: blockStmtNode, NodeType: "BlockStmt", NodeID: 3}, Parent: parentNode}
		parentNode.AddChild(blockStmtTreeNode)
		for i := 0; i < int(node.ChildCount()); i++ {
			buildQLTreeFromAST(node.Child(i), sourceCode, file, blockStmtTreeNode, storageNode)
		}
		return
	case "return_statement":
		returnStmtNode := javalang.ParseReturnStatement(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{ReturnStmt: returnStmtNode, NodeType: "ReturnStmt", NodeID: 4}, Parent: parentNode})
	case "assert_statement":
		assertStmtNode := javalang.ParseAssertStatement(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{AssertStmt: assertStmtNode, NodeType: "AssertStmt", NodeID: 5}, Parent: parentNode})
	case "yield_statement":
		yieldStmtNode := javalang.ParseYieldStatement(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{YieldStmt: yieldStmtNode, NodeType: "YieldStmt", NodeID: 6}, Parent: parentNode})
	case "break_statement":
		breakStmtNode := javalang.ParseBreakStatement(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{BreakStmt: breakStmtNode, NodeType: "BreakStmt", NodeID: 7}, Parent: parentNode})
	case "continue_statement":
		continueNode := javalang.ParseContinueStatement(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{ContinueStmt: continueNode, NodeType: "ContinueStmt", NodeID: 8}, Parent: parentNode})
	case "if_statement":
		IfNode := javalang.ParseIfStatement(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{IfStmt: IfNode, NodeType: "IfStmt", NodeID: 9}, Parent: parentNode})
	case "while_statement":
		whileStmtNode := javalang.ParseWhileStatement(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{WhileStmt: whileStmtNode, NodeType: "WhileStmt", NodeID: 10}, Parent: parentNode})
	case "do_statement":
		doWhileStmtNode := javalang.ParseDoWhileStatement(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{DoStmt: doWhileStmtNode, NodeType: "DoWhileStmt", NodeID: 11}, Parent: parentNode})
	case "for_statement":
		forStmtNode := javalang.ParseForLoopStatement(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{ForStmt: forStmtNode, NodeType: "ForStmt", NodeID: 12}, Parent: parentNode})
	case "binary_expression":
		binaryExprNode := javalang.ParseExpr(node, sourceCode, parentNode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{BinaryExpr: binaryExprNode, NodeType: "BinaryExpr", NodeID: 13}, Parent: parentNode})
		storageNode.AddBinaryExpr(binaryExprNode)
	case "method_declaration":
		methodDeclaration := javalang.ParseMethodDeclaration(node, sourceCode, file, parentNode)
		methodNode := &model.TreeNode{Node: &model.Node{MethodDecl: methodDeclaration, NodeType: "method_declaration", NodeID: 14}, Parent: parentNode}
		parentNode.AddChild(methodNode)
		storageNode.AddMethodDecl(methodDeclaration)
		for i := 0; i < int(node.ChildCount()); i++ {
			buildQLTreeFromAST(node.Child(i), sourceCode, file, methodNode, storageNode)
		}
		return
	case "method_invocation":
		methodInvokedNode := javalang.ParseMethodInvoker(node, sourceCode, file)
		methodInvocationTreeNode := &model.TreeNode{Node: &model.Node{MethodCall: methodInvokedNode, NodeType: "MethodCall", NodeID: 15}, Parent: parentNode}
		parentNode.AddChild(methodInvocationTreeNode)
		storageNode.AddMethodCall(methodInvokedNode)
		for i := 0; i < int(node.ChildCount()); i++ {
			buildQLTreeFromAST(node.Child(i), sourceCode, file, methodInvocationTreeNode, storageNode)
		}
		return
	case "class_declaration":
		classNode := javalang.ParseClass(node, sourceCode, file)
		classTreeNode := &model.TreeNode{Node: &model.Node{ClassDecl: classNode, NodeType: "ClassDeclaration", NodeID: 16}, Children: nil, Parent: parentNode}
		parentNode.AddChild(classTreeNode)
		storageNode.AddClassDecl(classNode)
		for i := 0; i < int(node.ChildCount()); i++ {
			buildQLTreeFromAST(node.Child(i), sourceCode, file, classTreeNode, storageNode)
		}
		return
	case "block_comment":
		// Parse block comments
		if strings.HasPrefix(node.Content(sourceCode), "/*") {
			javadocTags := javalang.ParseJavadocTags(node, sourceCode)
			parentNode.AddChild(&model.TreeNode{Node: &model.Node{JavaDoc: javadocTags, NodeType: "BlockComment", NodeID: 17}, Parent: parentNode})
		}
	case "local_variable_declaration", "field_declaration":
		// Extract variable name, type, and modifiers
		fieldNode := javalang.ParseField(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{Field: fieldNode, NodeType: "FieldDeclaration", NodeID: 18}, Children: nil, Parent: parentNode})
		storageNode.AddFieldDecl(fieldNode)
	case "object_creation_expression":
		classInstanceNode := javalang.ParseObjectCreationExpr(node, sourceCode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{ClassInstanceExpr: classInstanceNode, NodeType: "ObjectCreationExpr", NodeID: 19}, Children: nil, Parent: parentNode})
	}
	// Recursively process child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		buildQLTreeFromAST(node.Child(i), sourceCode, file, parentNode, storageNode)
	}
}

// Process a single file and return its tree.
func processFile(parser *sitter.Parser, file, fileName string, storageNode *db.StorageNode, workerID int, statusChan chan<- string) *model.TreeNode {
	sourceCode, err := readFile(file)
	if err != nil {
		utilities.Log("File not found:", err)
		return nil
	}

	// Parse the source code
	tree, err := parser.ParseCtx(context.TODO(), nil, sourceCode)
	if err != nil {
		utilities.Log("Error parsing file:", err)
		return nil
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	localTree := &model.TreeNode{
		Parent: nil,
		Node: &model.Node{
			FileNode: &model.File{File: fileName},
			NodeType: "File",
			NodeID:   20,
		},
	}

	statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Building graph and traversing code %s\033[0m", workerID, fileName)
	buildQLTreeFromAST(rootNode, sourceCode, file, localTree, storageNode)
	statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Done processing file %s\033[0m", workerID, fileName)

	return localTree
}

func getFiles(directory string) ([]string, error) {
	var files []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// append only java files
			if filepath.Ext(path) == ".java" {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

func readFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func Initialize(directory string, storageNode *db.StorageNode) []*model.TreeNode {
	treeHolder := []*model.TreeNode{}
	// record start time
	start := time.Now()

	files, err := getFiles(directory)
	if err != nil {
		//nolint:all
		utilities.Log("Directory not found:", err)
		return treeHolder
	}

	totalFiles := len(files)
	numWorkers := 5 // Number of concurrent workers
	fileChan := make(chan string, totalFiles)
	treeChan := make(chan *model.TreeNode, totalFiles)
	statusChan := make(chan string, numWorkers)
	progressChan := make(chan int, totalFiles)
	var wg sync.WaitGroup

	// Worker function
	worker := func(workerID int) {
		// Initialize the parser for each worker
		parser := sitter.NewParser()
		defer parser.Close()

		// Set the language (Java in this case)
		parser.SetLanguage(java.GetLanguage())

		for file := range fileChan {
			fileName := filepath.Base(file)
			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Reading and parsing code %s\033[0m", workerID, fileName)

			// Process file in a separate function to ensure proper cleanup
			localTree := processFile(parser, file, fileName, storageNode, workerID, statusChan)
			if localTree != nil {
				treeHolder = append(treeHolder, localTree)
				treeChan <- localTree
				progressChan <- 1
			}
		}
		wg.Done()
	}

	// Start workers
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(i + 1)
	}

	// Send files to workers
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// Status updater
	go func() {
		statusLines := make([]string, numWorkers)
		progress := 0
		for {
			select {
			case status, ok := <-statusChan:
				if !ok {
					return
				}
				workerID := int(status[12] - '0')
				statusLines[workerID-1] = status
			case _, ok := <-progressChan:
				if !ok {
					return
				}
				progress++
			}
			fmt.Print("\033[H\033[J") // Clear the screen
			for _, line := range statusLines {
				utilities.Log(line)
			}
			utilities.Fmt("Progress: %d%%\n", (progress*100)/totalFiles)
		}
	}()

	wg.Wait()
	close(statusChan)
	close(progressChan)
	close(treeChan)

	for _, packageDeclaration := range storageNode.Package {
		err := packageDeclaration.Insert(storageNode.DB)
		if err != nil {
			utilities.Log("Error inserting package:", err)
		}
	}
	for _, importDeclaration := range storageNode.ImportDecl {
		err := importDeclaration.Insert(storageNode.DB)
		if err != nil {
			utilities.Log("Error inserting import:", err)
		}
	}
	for _, classDeclaration := range storageNode.ClassDecl {
		err := classDeclaration.Insert(storageNode.DB)
		if err != nil {
			utilities.Log("Error inserting class:", err)
		}
	}
	for _, fieldDeclaration := range storageNode.Field {
		err := fieldDeclaration.Insert(storageNode.DB)
		if err != nil {
			utilities.Log("Error inserting field:", err)
		}
	}
	for _, methodDeclaration := range storageNode.MethodDecl {
		err := methodDeclaration.Insert(storageNode.DB)
		if err != nil {
			utilities.Log("Error inserting method:", err)
		}
	}
	for _, methodCallDeclaration := range storageNode.MethodCall {
		err := methodCallDeclaration.Insert(storageNode.DB)
		if err != nil {
			utilities.Log("Error inserting method call:", err)
		}
	}
	for _, binaryExpression := range storageNode.BinaryExpr {
		err := binaryExpression.Insert(storageNode.DB)
		if err != nil {
			utilities.Log("Error inserting binary expression:", err)
		}
	}

	for _, tree := range treeHolder {
		closureTableRows := []db.ClosureTableRow{}
		closureTableRows = db.BuildClosureTable(tree, []int64{}, 0, closureTableRows)
		db.StoreClosureTable(storageNode.DB, closureTableRows, tree.Node.FileNode.File)
	}

	storageNode.DB.Close()

	end := time.Now()
	elapsed := end.Sub(start)
	utilities.Log("Elapsed time: ", elapsed)
	utilities.Log("Project parsed successfully")

	return treeHolder
}
