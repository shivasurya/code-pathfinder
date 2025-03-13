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
            console.log(node.type);
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
        if (!data || !data.matches) return '';
        
        return data.matches.map(match => `
            <div class="match-item">
                <div class="match-type">${match.type}</div>
                <div class="match-name">${match.name || ''}</div>
                ${match.line ? `<div class="match-line">Line: ${match.line}</div>` : ''}
            </div>
        `).join('');
    }
}
