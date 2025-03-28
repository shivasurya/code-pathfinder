// AST Service - Handles AST processing and node management
export class ASTService {
    constructor() {
        this.validTypes = [
            'ClassDeclaration', 'ClassOrInterfaceDeclaration',
            'MethodDeclaration', 'ConstructorDeclaration',
            'VariableDeclaration', 'VariableDeclarator',
            'Parameter', 'LocalVariable',
            'FieldDeclaration', 'FieldAccess', 'MethodInvocation'
        ];
    }

    processNode(node, parentId = null, lastValidParentId = null, level = 0) {
        const nodeId = node.id || `node_${Math.random().toString(36).substr(2, 9)}`;
        let currentValidParentId = lastValidParentId;
        
        if (this.validTypes.includes(node.type)) {
            let category;
            let mass = 1; // Base mass for node physics

            if (node.type === 'ClassDeclaration' || node.type === 'ClassOrInterfaceDeclaration') {
                category = 'class';
                mass = 3; // Classes are heavier
            } else if (node.type === 'MethodDeclaration' || node.type === 'ConstructorDeclaration') {
                category = 'method';
                mass = 2; // Methods are medium weight
            } else if (node.type === 'FieldDeclaration' || node.type === 'FieldAccess') {
                category = 'fields';
                mass = 1.5; // Fields slightly more stable than variables
            } else if (node.type === 'MethodInvocation') {
                category = 'method-calls';
            }

            // Add node to visualization data
            this.visNodes.push({
                id: nodeId,
                label: `${node.type}\n${node.name || ''}`,
                type: node.type,
                category: category,
                level: level,
                mass: mass,
                name: node.name,
                line: node.line
            });

            // Connect to parent if exists
            if (parentId) {
                this.visEdges.push({
                    from: parentId,
                    to: nodeId,
                    arrows: 'to'
                });
            }

            currentValidParentId = nodeId;
        }

        // Process child nodes
        if (node.children) {
            node.children.forEach(child => {
                this.processNode(child, currentValidParentId, currentValidParentId, level + 1);
            });
        }
    }

    async parseAndVisualize(ast) {
        if (!ast) {
            console.error('No AST data provided');
            return { nodes: [], edges: [] };
        }

        try {
            this.visNodes = [];
            this.visEdges = [];
            this.processNode(ast);
            return { nodes: this.visNodes, edges: this.visEdges };
        } catch (error) {
            console.error('Error processing AST:', error);
            return { nodes: [], edges: [] };
        }
    }

    formatQueryResults(data) {
        if (!data || !data.results) return '';
        
        const tableHeader = `
            <tr class="results-table-header">
                <th>File</th>
                <th>Line</th>
                <th>Type</th>
            </tr>`;
            
        const tableRows = data.results.map(result => {
            const kind = result.kind || '';
            const category = this.getCategory(kind);
            
            return `
                <tr class="results-table-row">
                    <td class="file-cell" title="${result.file}">${result.file.split('/').pop()}</td>
                    <td class="line-cell">${result.line || '-'}</td>
                    <td class="kind-cell" data-category="${category}">${kind || '-'}</td>
                </tr>
            `;
        }).join('');
        
        return `
            <table class="results-table">
                ${tableHeader}
                ${tableRows}
            </table>
        `;
    }
    
    getCategory(kind) {
        // Java language-based types
        const javaTypes = [
            'method_declaration',
            'class_declaration',
            'interface_declaration',
            'field_declaration',
            'constructor_declaration',
            'variable_declaration',
            'parameter',
            'annotation',
            'enum_declaration',
            'package_declaration',
            'import_declaration'
        ];
        
        // Check if the kind matches any of our known types
        const kindLower = kind.toLowerCase();
        if (javaTypes.some(type => kindLower.includes(type))) return 'java';
        return 'other';
    }
}
