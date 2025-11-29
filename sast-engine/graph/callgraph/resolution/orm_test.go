package resolution

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestIsDjangoORMPattern(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		expectedMatch  bool
		expectedMethod string
	}{
		{
			name:           "Task.objects.filter",
			target:         "Task.objects.filter",
			expectedMatch:  true,
			expectedMethod: "filter",
		},
		{
			name:           "User.objects.get",
			target:         "User.objects.get",
			expectedMatch:  true,
			expectedMethod: "get",
		},
		{
			name:           "Annotation.objects.all",
			target:         "Annotation.objects.all",
			expectedMatch:  true,
			expectedMethod: "all",
		},
		{
			name:           "Model.objects.create",
			target:         "Model.objects.create",
			expectedMatch:  true,
			expectedMethod: "create",
		},
		{
			name:           "Task.objects without method",
			target:         "Task.objects",
			expectedMatch:  true,
			expectedMethod: "objects",
		},
		{
			name:           "Prefetch related",
			target:         "Article.objects.prefetch_related",
			expectedMatch:  true,
			expectedMethod: "prefetch_related",
		},
		{
			name:           "Select related",
			target:         "Post.objects.select_related",
			expectedMatch:  true,
			expectedMethod: "select_related",
		},
		{
			name:           "Instance method (not ORM)",
			target:         "task.save",
			expectedMatch:  false,
			expectedMethod: "",
		},
		{
			name:           "Regular function call",
			target:         "print",
			expectedMatch:  false,
			expectedMethod: "",
		},
		{
			name:           "Qualified model name",
			target:         "models.Task.objects.filter",
			expectedMatch:  true,
			expectedMethod: "filter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, method := IsDjangoORMPattern(tt.target)
			assert.Equal(t, tt.expectedMatch, match)
			assert.Equal(t, tt.expectedMethod, method)
		})
	}
}

func TestIsSQLAlchemyORMPattern(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		expectedMatch  bool
		expectedMethod string
	}{
		{
			name:           "User.query.filter",
			target:         "User.query.filter",
			expectedMatch:  true,
			expectedMethod: "filter",
		},
		{
			name:           "Model.query.all",
			target:         "Model.query.all",
			expectedMatch:  true,
			expectedMethod: "all",
		},
		{
			name:           "User.query.filter_by",
			target:         "User.query.filter_by",
			expectedMatch:  true,
			expectedMethod: "filter_by",
		},
		{
			name:           "Not SQLAlchemy pattern",
			target:         "task.save",
			expectedMatch:  false,
			expectedMethod: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, method := IsSQLAlchemyORMPattern(tt.target)
			assert.Equal(t, tt.expectedMatch, match)
			assert.Equal(t, tt.expectedMethod, method)
		})
	}
}

func TestIsORMPattern(t *testing.T) {
	tests := []struct {
		name            string
		target          string
		expectedMatch   bool
		expectedORM     string
		expectedMethod  string
	}{
		{
			name:           "Django ORM filter",
			target:         "Task.objects.filter",
			expectedMatch:  true,
			expectedORM:    "Django ORM",
			expectedMethod: "filter",
		},
		{
			name:           "SQLAlchemy query",
			target:         "User.query.all",
			expectedMatch:  true,
			expectedORM:    "SQLAlchemy",
			expectedMethod: "all",
		},
		{
			name:           "Not an ORM pattern",
			target:         "helper.process",
			expectedMatch:  false,
			expectedORM:    "",
			expectedMethod: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, orm, method := IsORMPattern(tt.target)
			assert.Equal(t, tt.expectedMatch, match)
			assert.Equal(t, tt.expectedORM, orm)
			assert.Equal(t, tt.expectedMethod, method)
		})
	}
}

func TestValidateDjangoModel(t *testing.T) {
	// Create mock code graph with some class definitions
	nodes := make(map[string]*graph.Node)
	nodeList := []*graph.Node{
		{
			Type:       "class_declaration",
			Name:       "Task",
			SuperClass: "models.Model",
		},
		{
			Type:       "class_declaration",
			Name:       "User",
			SuperClass: "Model",
		},
		{
			Type:       "class_declaration",
			Name:       "Article",
			SuperClass: "BaseModel",
		},
		{
			Type:       "class_declaration",
			Name:       "TaskView",
			SuperClass: "View",
		},
		{
			Type:       "class_declaration",
			Name:       "TaskSerializer",
			SuperClass: "Serializer",
		},
	}

	for _, node := range nodeList {
		nodes[node.ID] = node
	}

	codeGraph := &graph.CodeGraph{Nodes: nodes}

	tests := []struct {
		name      string
		modelName string
		expected  bool
	}{
		{
			name:      "Task with models.Model superclass",
			modelName: "Task",
			expected:  true,
		},
		{
			name:      "User with Model superclass",
			modelName: "User",
			expected:  true,
		},
		{
			name:      "Article (PascalCase, not View/Serializer)",
			modelName: "Article",
			expected:  true,
		},
		{
			name:      "TaskView (has View suffix)",
			modelName: "TaskView",
			expected:  false,
		},
		{
			name:      "TaskSerializer (has Serializer suffix)",
			modelName: "TaskSerializer",
			expected:  false,
		},
		{
			name:      "Unknown PascalCase name",
			modelName: "Annotation",
			expected:  true, // Heuristic: PascalCase likely a model
		},
		{
			name:      "lowercase name",
			modelName: "task",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateDjangoModel(tt.modelName, codeGraph)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveDjangoORMCall(t *testing.T) {
	registry := core.NewModuleRegistry()
	nodes := make(map[string]*graph.Node)
	nodeList := []*graph.Node{
		{
			Type:       "class_declaration",
			Name:       "Task",
			SuperClass: "models.Model",
		},
	}

	for _, node := range nodeList {
		nodes[node.ID] = node
	}

	codeGraph := &graph.CodeGraph{Nodes: nodes}

	tests := []struct {
		name         string
		target       string
		modulePath   string
		expectedFQN  string
		expectedBool bool
	}{
		{
			name:         "Task.objects.filter",
			target:       "Task.objects.filter",
			modulePath:   "myapp.models",
			expectedFQN:  "myapp.models.Task.objects.filter",
			expectedBool: true,
		},
		{
			name:         "User.objects.get",
			target:       "User.objects.get",
			modulePath:   "auth.models",
			expectedFQN:  "auth.models.User.objects.get",
			expectedBool: true,
		},
		{
			name:         "Annotation.objects.all",
			target:       "Annotation.objects.all",
			modulePath:   "app.models",
			expectedFQN:  "app.models.Annotation.objects.all",
			expectedBool: true,
		},
		{
			name:         "Not ORM pattern",
			target:       "helper.process",
			modulePath:   "utils",
			expectedFQN:  "helper.process",
			expectedBool: false,
		},
		{
			name:         "Qualified model name",
			target:       "models.Task.objects.filter",
			modulePath:   "myapp.views",
			expectedFQN:  "myapp.views.Task.objects.filter",
			expectedBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fqn, resolved := ResolveDjangoORMCall(tt.target, tt.modulePath, registry, codeGraph)
			assert.Equal(t, tt.expectedFQN, fqn)
			assert.Equal(t, tt.expectedBool, resolved)
		})
	}
}

func TestResolveSQLAlchemyORMCall(t *testing.T) {
	tests := []struct {
		name         string
		target       string
		modulePath   string
		expectedFQN  string
		expectedBool bool
	}{
		{
			name:         "User.query.filter",
			target:       "User.query.filter",
			modulePath:   "models",
			expectedFQN:  "models.User.query.filter",
			expectedBool: true,
		},
		{
			name:         "Model.query.all",
			target:       "Model.query.all",
			modulePath:   "app.models",
			expectedFQN:  "app.models.Model.query.all",
			expectedBool: true,
		},
		{
			name:         "Not SQLAlchemy pattern",
			target:       "helper.process",
			modulePath:   "utils",
			expectedFQN:  "helper.process",
			expectedBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fqn, resolved := ResolveSQLAlchemyORMCall(tt.target, tt.modulePath)
			assert.Equal(t, tt.expectedFQN, fqn)
			assert.Equal(t, tt.expectedBool, resolved)
		})
	}
}

func TestResolveORMCall(t *testing.T) {
	registry := core.NewModuleRegistry()
	codeGraph := &graph.CodeGraph{Nodes: make(map[string]*graph.Node)}

	tests := []struct {
		name         string
		target       string
		modulePath   string
		expectedFQN  string
		expectedBool bool
	}{
		{
			name:         "Django ORM",
			target:       "Task.objects.filter",
			modulePath:   "myapp.models",
			expectedFQN:  "myapp.models.Task.objects.filter",
			expectedBool: true,
		},
		{
			name:         "SQLAlchemy ORM",
			target:       "User.query.all",
			modulePath:   "models",
			expectedFQN:  "models.User.query.all",
			expectedBool: true,
		},
		{
			name:         "Not ORM pattern",
			target:       "helper.process",
			modulePath:   "utils",
			expectedFQN:  "helper.process",
			expectedBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fqn, resolved := ResolveORMCall(tt.target, tt.modulePath, registry, codeGraph)
			assert.Equal(t, tt.expectedFQN, fqn)
			assert.Equal(t, tt.expectedBool, resolved)
		})
	}
}
