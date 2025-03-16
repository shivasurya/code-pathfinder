// Visualization Service - Handles graph visualization using vis.js
export class VisualizationService {
    constructor() {
        this.network = null;
        this.currentNodes = [];
        this.currentEdges = [];
        this.visNodes = [];
        this.visEdges = [];
        this.options = {
            nodes: {
                shape: 'box',
                size: 16,
                borderWidth: 2,
                color: {
                    border: '#61dafb',
                    background: '#1e1e1e'
                },
                font: {
                    face: 'Inter',
                    size: 14,
                    color: '#ffffff'
                },
                shadow: {
                    enabled: true,
                    color: 'rgba(97, 218, 251, 0.2)',
                    size: 4,
                    x: 0,
                    y: 2
                }
            },
            edges: {
                color: {
                    color: '#4d4d4d',
                    opacity: 0.6,
                    highlight: '#61dafb',
                    hover: '#61dafb'
                },
                width: 2,
                smooth: {
                    type: 'curvedCW',
                    roundness: 0.2,
                    forceDirection: 'none'
                },
                arrows: {
                    to: {
                        enabled: true,
                        scaleFactor: 0.7,
                        type: 'circle'
                    }
                },
                shadow: {
                    enabled: true,
                    color: 'rgba(0,0,0,0.2)',
                    size: 5,
                    x: 2,
                    y: 2
                }
            },
            physics: {
                enabled: true,
                solver: 'barnesHut',
                barnesHut: {
                    gravitationalConstant: -10000,
                    centralGravity: 1.5,
                    springLength: 300,
                    springConstant: 0.5,
                    damping: 0.9,
                    avoidOverlap: 1
                },
                stabilization: {
                    enabled: true,
                    iterations: 1000,
                    updateInterval: 10,
                    onlyDynamicEdges: false,
                    fit: true
                },
                minVelocity: 0.01,
                maxVelocity: 10,
                timestep: 0.1,
                adaptiveTimestep: true
            },
            layout: {
                improvedLayout: true,
                randomSeed: 42,
                hierarchical: {
                    enabled: false,
                    direction: 'LR',
                    sortMethod: 'directed',
                    nodeSpacing: 200,
                    levelSeparation: 300
                }
            },
            groups: this.getNodeGroups()
        };
    }

    getNodeGroups() {
        return {
            class: {
                color: { 
                    background: '#4CAF50',
                    border: '#66BB6A',
                    highlight: { background: '#66BB6A', border: '#66BB6A' },
                    hover: { background: '#66BB6A', border: '#66BB6A' }
                }
            },
            constructordeclaration: {
                background: '#2196F3',
                border: '#2196F3',
                highlight: { background: '#42A5F5', border: '#42A5F5' },
                hover: { background: '#42A5F5', border: '#42A5F5' }
            },
            'MethodDeclaration': {
                background: '#2196F3',
                border: '#2196F3',
                highlight: { background: '#42A5F5', border: '#42A5F5' },
                hover: { background: '#42A5F5', border: '#42A5F5' }
            },
            methodinvocation: {
                background: '#9C27B0',
                border: '#9C27B0',
                highlight: { background: '#AB47BC', border: '#AB47BC' },
                hover: { background: '#AB47BC', border: '#AB47BC' }
            }
        };
    }

    initializeNetwork(container) {
        if (!container) return;

        this.network = new vis.Network(container, {
            nodes: new vis.DataSet([]),
            edges: new vis.DataSet([])
        }, this.options);

        this.initializeNetworkEvents();
        return this.network;
    }

    initializeNetworkEvents() {
        if (!this.network) return;

        this.network.on('click', (params) => {
            if (params.nodes.length > 0) {
                const nodeId = params.nodes[0];
                const node = this.currentNodes.find(n => n.id === nodeId);
                if (node) {
                    console.log('Selected node:', node);
                }
            }
        });
    }

    updateVisualization(newNodes = [], newEdges = []) {
        if (!this.network || !newNodes || !newEdges) return;
        
        this.currentNodes = newNodes;
        this.currentEdges = newEdges;
        this.visNodes = newNodes;
        this.visEdges = newEdges;

        const nodesDataSet = new vis.DataSet(newNodes.map(node => ({
            id: node.id,
            label: node.label || `${node.type}\n${node.name || ''}`,
            color: this.getNodeColor(node.type),
            font: {
                color: '#ffffff',
                size: 14,
                face: 'Inter'
            },
            shape: 'box',
            margin: 10,
            shadow: true,
            title: node.line ? `Line: ${node.line}` : undefined
        })));

        const edgesDataSet = new vis.DataSet(newEdges.map(edge => ({
            from: edge.from,
            to: edge.to,
            arrows: 'to',
            color: { color: '#4d4d4d', highlight: '#61dafb' },
            width: 1,
            smooth: {
                type: 'continuous',
                roundness: 0.2
            }
        })));

        this.network.setData({ nodes: nodesDataSet, edges: edgesDataSet });
    }

    getNodeColor(type) {
        const nodeType = type.toLowerCase();
        const colors = {
            classdeclaration: { background: '#4CAF50', border: '#66BB6A' },
            methoddeclaration: { background: '#2196F3', border: '#2196F3' },
            fielddeclaration: { background: '#FF9800', border: '#FF9800' },
            compilationunit: { background: '#ec8c4c', border: '#c678dd' },
            constructordeclaration: { background: '#2196F3', border: '#2196F3' },
            methodinvocation: { background: '#9C27B0', border: '#AB47BC' },
            default: { background: '#FF5722', border: '#FF5722' }
        };

        if (nodeType.includes('class')) return colors.classdeclaration;
        if (nodeType.includes('methoddeclaration')) return colors.methoddeclaration;
        if (nodeType.includes('field')) return colors.fielddeclaration;
        if (nodeType.includes('compilation')) return colors.compilationunit;
        if (nodeType.includes('methodinvocation')) return colors.methodinvocation;
        if (nodeType.includes('constructor')) return colors.constructordeclaration;
        return colors.default;
    }
}
