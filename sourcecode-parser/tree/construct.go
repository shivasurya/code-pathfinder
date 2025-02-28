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
		importDeclNode := javalang.ParseImportDeclaration(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{ImportType: importDeclNode}, Parent: parentNode})
		storageNode.AddImportDecl(importDeclNode)
	case "package_declaration":
		packageDeclNode := javalang.ParsePackageDeclaration(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{Package: packageDeclNode}, Parent: parentNode})
		storageNode.AddPackage(packageDeclNode)
	case "block":
		blockStmtNode := javalang.ParseBlockStatement(node, sourceCode, file)
		blockStmtTreeNode := &model.TreeNode{Node: &model.Node{BlockStmt: blockStmtNode}, Parent: parentNode}
		parentNode.AddChild(blockStmtTreeNode)
		for i := 0; i < int(node.ChildCount()); i++ {
			buildQLTreeFromAST(node.Child(i), sourceCode, file, blockStmtTreeNode, storageNode)
		}
		return
	case "return_statement":
		returnStmtNode := javalang.ParseReturnStatement(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{ReturnStmt: returnStmtNode}, Parent: parentNode})
	case "assert_statement":
		assertStmtNode := javalang.ParseAssertStatement(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{AssertStmt: assertStmtNode}, Parent: parentNode})
	case "yield_statement":
		yieldStmtNode := javalang.ParseYieldStatement(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{YieldStmt: yieldStmtNode}, Parent: parentNode})
	case "break_statement":
		breakStmtNode := javalang.ParseBreakStatement(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{BreakStmt: breakStmtNode}, Parent: parentNode})
	case "continue_statement":
		continueNode := javalang.ParseContinueStatement(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{ContinueStmt: continueNode}, Parent: parentNode})
	case "if_statement":
		IfNode := javalang.ParseIfStatement(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{IfStmt: IfNode}, Parent: parentNode})
	case "while_statement":
		whileStmtNode := javalang.ParseWhileStatement(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{WhileStmt: whileStmtNode}, Parent: parentNode})
	case "do_statement":
		doWhileStmtNode := javalang.ParseDoWhileStatement(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{DoStmt: doWhileStmtNode}, Parent: parentNode})
	case "for_statement":
		forStmtNode := javalang.ParseForLoopStatement(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{ForStmt: forStmtNode}, Parent: parentNode})
	case "binary_expression":
		binaryExprNode := javalang.ParseExpr(node, sourceCode, file, parentNode)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{BinaryExpr: binaryExprNode}, Parent: parentNode})
		storageNode.AddBinaryExpr(binaryExprNode)
	case "method_declaration":
		methodDeclaration := javalang.ParseMethodDeclaration(node, sourceCode, file)
		utilities.Log("Processing method:", methodDeclaration.Name, "in file:", file)
		methodNode := &model.TreeNode{Node: &model.Node{MethodDecl: methodDeclaration}, Parent: parentNode}
		parentNode.AddChild(methodNode)
		storageNode.AddMethodDecl(methodDeclaration)
		for i := 0; i < int(node.ChildCount()); i++ {
			buildQLTreeFromAST(node.Child(i), sourceCode, file, methodNode, storageNode)
		}
		return
	case "method_invocation":
		methodInvokedNode := javalang.ParseMethodInvoker(node, sourceCode, file)
		methodInvocationTreeNode := &model.TreeNode{Node: &model.Node{MethodCall: methodInvokedNode}, Parent: parentNode}
		parentNode.AddChild(methodInvocationTreeNode)
		storageNode.AddMethodCall(methodInvokedNode)
		for i := 0; i < int(node.ChildCount()); i++ {
			buildQLTreeFromAST(node.Child(i), sourceCode, file, methodInvocationTreeNode, storageNode)
		}
		return
	case "class_declaration":
		classNode := javalang.ParseClass(node, sourceCode, file)
		classTreeNode := &model.TreeNode{Node: &model.Node{ClassDecl: classNode}, Children: nil, Parent: parentNode}
		parentNode.AddChild(classTreeNode)
		storageNode.AddClassDecl(classNode)
		for i := 0; i < int(node.ChildCount()); i++ {
			buildQLTreeFromAST(node.Child(i), sourceCode, file, classTreeNode, storageNode)
		}
		return
	case "block_comment":
		// Parse block comments
		if strings.HasPrefix(node.Content(sourceCode), "/*") {
			javadocTags := javalang.ParseJavadocTags(node, sourceCode, file)
			parentNode.AddChild(&model.TreeNode{Node: &model.Node{JavaDoc: javadocTags}, Parent: parentNode})
		}
	case "local_variable_declaration", "field_declaration":
		// Extract variable name, type, and modifiers
		fieldNode := javalang.ParseField(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{Field: fieldNode}, Children: nil, Parent: parentNode})
	case "object_creation_expression":
		classInstanceNode := javalang.ParseObjectCreationExpr(node, sourceCode, file)
		parentNode.AddChild(&model.TreeNode{Node: &model.Node{ClassInstanceExpr: classInstanceNode}, Children: nil, Parent: parentNode})
	}
	// Recursively process child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		buildQLTreeFromAST(node.Child(i), sourceCode, file, parentNode, storageNode)
	}
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

func Initialize(directory string) []*model.TreeNode {
	treeHolder := []*model.TreeNode{}
	storageNode := db.NewStorageNode(directory)
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
			sourceCode, err := readFile(file)
			if err != nil {
				utilities.Log("File not found:", err)
				continue
			}
			// Parse the source code
			tree, err := parser.ParseCtx(context.TODO(), nil, sourceCode)
			if err != nil {
				utilities.Log("Error parsing file:", err)
				continue
			}
			//nolint:all
			defer tree.Close()

			rootNode := tree.RootNode()
			localTree := &model.TreeNode{
				Parent: nil,
				Node: &model.Node{
					FileNode: &model.File{File: fileName},
				},
			}
			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Building graph and traversing code %s\033[0m", workerID, fileName)
			buildQLTreeFromAST(rootNode, sourceCode, file, localTree, storageNode)
			treeHolder = append(treeHolder, localTree)
			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Done processing file %s\033[0m", workerID, fileName)

			treeChan <- localTree
			progressChan <- 1
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
		packageDeclaration.Insert(storageNode.DB)
	}
	for _, importDeclaration := range storageNode.ImportDecl {
		importDeclaration.Insert(storageNode.DB)
	}
	//TODO: class decl, field, method call, binary expr pending
	for _, classDeclaration := range storageNode.ClassDecl {
		classDeclaration.Insert(storageNode.DB)
	}
	for _, fieldDeclaration := range storageNode.Field {
		fieldDeclaration.Insert(storageNode.DB)
	}
	for _, methodDeclaration := range storageNode.MethodDecl {
		methodDeclaration.Insert(storageNode.DB)
	}
	for _, methodCallDeclaration := range storageNode.MethodCall {
		methodCallDeclaration.Insert(storageNode.DB)
	}
	for _, binaryExpression := range storageNode.BinaryExpr {
		binaryExpression.Insert(storageNode.DB)
	}

	storageNode.DB.Close()

	end := time.Now()
	elapsed := end.Sub(start)
	utilities.Log("Elapsed time: ", elapsed)
	utilities.Log("Project parsed successfully")

	return treeHolder
}
