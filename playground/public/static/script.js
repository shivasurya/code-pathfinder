// Debounce function to limit the rate of API calls
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Initialize split panel resizing
function initResizablePanel() {
    const gutter = document.querySelector('.gutter');
    const leftPanel = document.querySelector('.left-panel');
    const rightPanel = document.querySelector('.right-panel');
    let isResizing = false;
    let startX;
    let startLeftWidth;

    gutter.addEventListener('mousedown', (e) => {
        isResizing = true;
        gutter.classList.add('active');
        startX = e.pageX;
        startLeftWidth = leftPanel.offsetWidth;
    });

    document.addEventListener('mousemove', (e) => {
        if (!isResizing) return;

        const mainContent = document.querySelector('.main-content');
        const totalWidth = mainContent.offsetWidth;
        const dx = e.pageX - startX;
        
        // Calculate new widths as percentages
        let newLeftWidth = ((startLeftWidth + dx) / totalWidth) * 100;
        newLeftWidth = Math.min(Math.max(newLeftWidth, 20), 80); // Limit between 20% and 80%
        
        leftPanel.style.width = `${newLeftWidth}%`;
        rightPanel.style.width = `${100 - newLeftWidth}%`;
        gutter.style.left = `${newLeftWidth}%`;

        // Refresh editor and visualization
        editor.refresh();
        updateVisualization(currentNodes, currentEdges);
    });

    document.addEventListener('mouseup', () => {
        isResizing = false;
        gutter.classList.remove('active');
    });
}

// UI State Management
const UIState = {
    activeTab: 'visualization',
    setActiveTab(tabName) {
        this.activeTab = tabName;
        this.updateTabUI();
    },
    updateTabUI() {
        document.querySelectorAll('.tab-button').forEach(button => {
            button.classList.toggle('active', button.dataset.tab === this.activeTab);
        });
        document.querySelectorAll('.tab-pane').forEach(pane => {
            pane.classList.toggle('active', pane.id === `${this.activeTab}-tab`);
        });
    }
};

// Network Visualization Service
const VisualizationService = {
    updateVisualization(nodes, edges) {
        if (!network) return;

        const data = {
            nodes: new vis.DataSet(nodes),
            edges: new vis.DataSet(edges)
        };

        network.setData(data);
        currentNodes = nodes;
        currentEdges = edges;

        // Fit the network view
        network.fit({
            animation: {
                duration: 1000,
                easingFunction: 'easeInOutQuad'
            }
        });
    },

    initializeNetwork(container) {
        network = new vis.Network(container, {
            nodes: new vis.DataSet([]),
            edges: new vis.DataSet([])
        }, options);

        this.initializeNetworkEvents();
    },

    initializeNetworkEvents() {
        if (!network) return;

        network.on('click', (params) => {
            if (params.nodes.length > 0) {
                const nodeId = params.nodes[0];
                const node = currentNodes.find(n => n.id === nodeId);
                if (node) {
                    console.log('Selected node:', node);
                }
            }
        });

        network.on('stabilizationProgress', (params) => {
            console.log('Layout stabilization:', Math.round(params.iterations / params.total * 100), '%');
        });

        network.on('stabilizationIterationsDone', () => {
            console.log('Layout stabilization finished');
        });
    }
};

// Initialize global variables
let network = null;
let editor = null;
let queryEditor = null;
let currentNodes = [];
let currentEdges = [];

// Network visualization options
const options = {
    nodes: {
        shape: 'circle',
        margin: 10,
        widthConstraint: {
            maximum: 200
        },
        borderWidth: 2,
        color: {
            border: '#61dafb',
            background: '#1e1e1e'
        },
        font: {
            face: 'Inter',
            size: 14,
            color: '#ffffff',
            multi: true,
            bold: {
                color: '#61dafb',
                size: 15
            }
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
            highlight: '#61dafb',
            hover: '#61dafb'
        },
        width: 1.5,
        smooth: {
            type: 'cubicBezier',
            forceDirection: 'vertical',
            roundness: 0.3
        },
        arrows: {
            to: {
                enabled: true,
                scaleFactor: 0.8,
                type: 'arrow'
            }
        },
        selectionWidth: 2,
        hoverWidth: 2
    },
    physics: {
        enabled: true,
        hierarchicalRepulsion: {
            nodeDistance: 150,
            springLength: 150,
            springConstant: 0.2,
            damping: 0.09
        },
        stabilization: {
            enabled: true,
            iterations: 1000,
            updateInterval: 50,
            fit: true
        }
    },
    interaction: {
        hover: true,
        tooltipDelay: 200,
        zoomView: true,
        dragView: true
    },
    layout: {
        improvedLayout: true,
        hierarchical: {
            enabled: true,
            direction: 'UD',
            sortMethod: 'directed',
            nodeSpacing: 120,
            levelSeparation: 150,
            blockShifting: true,
            edgeMinimization: true,
            parentCentralization: true,
            treeSpacing: 100
        }
    }
};

// Initialize application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    initResizablePanel();

    const container = document.getElementById('visualization');
    if (container) {
        VisualizationService.initializeNetwork(container);
    }

    // Initialize editors
    editor = CodeMirror(document.getElementById('codeEditor'), {
        mode: 'text/x-java',
        theme: 'monokai',
        lineNumbers: true,
        lineWrapping: true,
        scrollbarStyle: 'native',
        viewportMargin: Infinity,
        value: `public class UserService {
    private final UserRepository userRepository;
    private final Logger logger;

    public UserService(UserRepository userRepository) {
        this.userRepository = userRepository;
        this.logger = LoggerFactory.getLogger(UserService.class);
    }

    public User getUserById(String id) {
        logger.info("Fetching user with id: {}", id);
        return userRepository.findById(id)
            .orElseThrow(() -> new UserNotFoundException("User not found"));
    }

    public List<User> getAllUsers() {
        return userRepository.findAll();
    }

    public User createUser(User user) {
        if (userRepository.existsByEmail(user.getEmail())) {
            throw new DuplicateEmailException("Email already exists");
        }
        return userRepository.save(user);
    }

    public void deleteUser(String id) {
        if (!userRepository.existsById(id)) {
            throw new UserNotFoundException("User not found");
        }
        userRepository.deleteById(id);
    }
}`
    });

    queryEditor = CodeMirror(document.getElementById('queryEditor'), {
        mode: 'text/x-java',
        theme: 'monokai',
        lineNumbers: true,
        lineWrapping: true,
        placeholder: 'Enter your query here...'
    });

    document.getElementById('executeQuery').addEventListener('click', async () => {
        const code = editor.getValue();
        const query = queryEditor.getValue();
        await executeQuery(code, query);
        UIState.setActiveTab('results');
    });

    // Initialize tab switching
    document.querySelectorAll('.tab-button').forEach(button => {
        button.addEventListener('click', () => {
            UIState.setActiveTab(button.dataset.tab);
        });
    });

    // Add automatic parsing on code change
    editor.on('change', debounce(() => {
        const code = editor.getValue();
        //ASTService.parseAndVisualize(code);
    }, 1000));
});

// Create zoom controls for the visualization
const zoomControls = document.createElement('div');
zoomControls.className = 'zoom-controls';
zoomControls.innerHTML = `
    <button class="zoom-in">+</button>
    <button class="zoom-out">−</button>
    <button class="zoom-reset">⟲</button>
    <span class="zoom-level">100%</span>
`;

// Initialize visualization when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    // Initialize resizable panels
    initResizablePanel();

    // Initialize network container
    const container = document.getElementById('visualization');
    if (container) {
        // Create the network
        network = new vis.Network(container, {
            nodes: new vis.DataSet([]),
            edges: new vis.DataSet([])
        }, options);

        // Add zoom controls to the container
        container.appendChild(zoomControls);

        // Add zoom control event listeners
        document.querySelector('.zoom-in')?.addEventListener('click', () => {
            if (network) {
                network.moveTo({
                    scale: network.getScale() * 1.2
                });
            }
        });

        document.querySelector('.zoom-out')?.addEventListener('click', () => {
            if (network) {
                network.moveTo({
                    scale: network.getScale() * 0.8
                });
            }
        });

        document.querySelector('.zoom-reset')?.addEventListener('click', () => {
            if (network) {
                network.fit();
            }
        });
    }

    // Add event listeners for buttons
    document.getElementById('executeQuery')?.addEventListener('click', async () => {
        const javaSource = editor.getValue();
        const query = queryEditor.getValue();

        try {
            // Execute query
            const response = await fetch('/api/analyze', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    javaSource,
                    query
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            document.getElementById('queryResults').innerHTML = formatQueryResults(data);
            highlightNodes(data.matches);
        } catch (error) {
            console.error('Error:', error);
            document.getElementById('queryResults').innerHTML = 
                `<span style="color: #ff6b6b;">Error: ${error.message}</span>`;
        }
    });

    document.getElementById('parseAST')?.addEventListener('click', async () => {
        const javaSource = editor.getValue();

        try {
            // Parse AST
            const response = await fetch('/api/parse', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    code: javaSource
                })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            const visNodes = [];
            const visEdges = [];

            // Process AST nodes
            function processNode(node, parentId = null, lastValidParentId = null) {
                const validTypes = ['ClassDeclaration', 'ClassOrInterfaceDeclaration',
                                  'MethodDeclaration', 'ConstructorDeclaration',
                                  'VariableDeclaration', 'VariableDeclarator',
                                  'Parameter', 'LocalVariable',
                                  'FieldDeclaration', 'FieldAccess'];

                const nodeId = node.id || `node_${Math.random().toString(36).substr(2, 9)}`;
                let currentValidParentId = lastValidParentId;

                // Determine node category and add to visualization if it's a valid type
                if (validTypes.includes(node.type)) {
                    let category = 'expressions';

                    if (node.type === 'ClassDeclaration' || node.type === 'ClassOrInterfaceDeclaration') {
                        category = 'class';
                    } else if (node.type === 'MethodDeclaration' || node.type === 'ConstructorDeclaration') {
                        category = 'constructor-method';
                    } else if (node.type === 'VariableDeclaration' || node.type === 'VariableDeclarator' || 
                            node.type === 'Parameter' || node.type === 'LocalVariable') {
                        category = 'variables';
                    } else if (node.type === 'FieldDeclaration' || node.type === 'FieldAccess') {
                        category = 'fields';
                    }

                    visNodes.push({
                        id: nodeId,
                        label: node.name || node.type,
                        type: node.type,
                        group: category
                    });

                    // Connect to the last valid parent if it exists
                    if (lastValidParentId) {
                        visEdges.push({
                            from: lastValidParentId,
                            to: nodeId,
                            arrows: 'to'
                        });
                    }

                    // Update the last valid parent ID for children
                    currentValidParentId = nodeId;
                }

                // Process children with the current valid parent ID
                if (node.children) {
                    node.children.forEach(child => processNode(child, nodeId, currentValidParentId));
                }
            }

            processNode(data.ast);
            updateVisualization(visNodes, visEdges);
            updateASTList(visNodes);
        } catch (error) {
            console.error('Error:', error);
            document.getElementById('astList').innerHTML = 
                `<span style="color: #ff6b6b;">Error parsing AST: ${error.message}</span>`;
        }
    });
});

// Format query results
function formatQueryResults(data) {
    if (!data.matches || data.matches.length === 0) {
        return '<span style="color: #ffd700;">No matches found</span>';
    }

    return `<span style="color: #98c379;">Found ${data.matches.length} matches:</span>\n\n` +
        data.matches.map(match => {
            return `• ${match.type}: ${match.label}\n  ${match.details || ''}`;
        }).join('\n\n');
}

// Highlight matching nodes in the visualization
function highlightNodes(matches) {
    if (!matches || !matches.length || !network) return;

    const matchIds = new Set(matches.map(m => m.id));
    const allNodes = network.body.data.nodes.get();
    const allEdges = network.body.data.edges.get();

    // Update nodes
    allNodes.forEach(node => {
        const isHighlighted = matchIds.has(node.id);
        network.body.data.nodes.update({
            id: node.id,
            opacity: isHighlighted ? 1 : 0.2,
            font: {
                ...node.font,
                color: isHighlighted ? '#ffffff' : 'rgba(255,255,255,0.3)'
            }
        });
    });

    // Update edges
    allEdges.forEach(edge => {
        const isHighlighted = matchIds.has(edge.from) && matchIds.has(edge.to);
        network.body.data.edges.update({
            id: edge.id,
            opacity: isHighlighted ? 0.8 : 0.1
        });
    });
}

function updateVisualization(newNodes = [], newEdges = []) {
    if (!network || !newNodes || !newEdges) return;
    
    // Update current graph data
    currentNodes = newNodes;
    currentEdges = newEdges;

    // Create DataSet for nodes with proper styling
    const nodesDataSet = new vis.DataSet(newNodes.map(node => ({
        id: node.id,
        label: node.label || `${node.type}\n${node.name || ''}`,
        color: getNodeColor(node.type),
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

    // Create DataSet for edges with consistent styling
    const edgesDataSet = new vis.DataSet(newEdges.map(edge => ({
        from: edge.from,
        to: edge.to,
        arrows: 'to',
        color: { color: '#4d4d4d', highlight: '#61dafb' },
        width: 1,
        smooth: {
            type: 'continuous',
            roundness: 0.5
        }
    })));

    // Update the network
    network.setData({
        nodes: nodesDataSet,
        edges: edgesDataSet
    });

    // Stabilize and fit the network
    network.stabilize();
    network.fit();
}

function getNodeColor(type) {
    if (!type) return '#FF5722';

    const colors = {
        'classdeclaration': {
            background: '#4CAF50',
            border: '#4CAF50',
            highlight: { background: '#66BB6A', border: '#66BB6A' },
            hover: { background: '#66BB6A', border: '#66BB6A' }
        },
        'methoddeclaration': {
            background: '#2196F3',
            border: '#2196F3',
            highlight: { background: '#42A5F5', border: '#42A5F5' },
            hover: { background: '#42A5F5', border: '#42A5F5' }
        },
        'fielddeclaration': {
            background: '#FF9800',
            border: '#FF9800',
            highlight: { background: '#FFA726', border: '#FFA726' },
            hover: { background: '#FFA726', border: '#FFA726' }
        },
        'compilationunit': {
            background: '#9C27B0',
            border: '#9C27B0',
            highlight: { background: '#AB47BC', border: '#AB47BC' },
            hover: { background: '#AB47BC', border: '#AB47BC' }
        },
        'default': {
            background: '#FF5722',
            border: '#FF5722',
            highlight: { background: '#FF7043', border: '#FF7043' },
            hover: { background: '#FF7043', border: '#FF7043' }
        }
    };

    const nodeType = type.toLowerCase();
    if (nodeType.includes('class')) return colors.classdeclaration;
    if (nodeType.includes('method')) return colors.methoddeclaration;
    if (nodeType.includes('field')) return colors.fielddeclaration;
    if (nodeType.includes('compilation')) return colors.compilationunit;
    return colors.default;
}

function updateASTList(nodes) {
    const astList = document.querySelector('.ast-list');
    if (!astList) return; // Skip if element doesn't exist
    
    astList.innerHTML = nodes.map(node => `
        <div class="ast-item">
            <span class="ast-type">${node.type}</span>
            <span class="ast-label">${node.label}</span>
        </div>
    `).join('');
}

// Initialize network event handlers
function initializeNetworkEvents() {
    if (!network) return;

    // Handle node selection
    network.on('selectNode', function(params) {
        if (params.nodes.length > 0) {
            const nodeId = params.nodes[0];
            const node = currentNodes.find(n => n.id === nodeId);
            if (node) {
                const details = document.getElementById('nodeDetails');
                details.innerHTML = `
                    <h3>Node Details</h3>
                    <p>Type: ${node.type}</p>
                    <p>Line: ${node.line || 'N/A'}</p>
                    ${node.name ? `<p>Name: ${node.name}</p>` : ''}
                `;
                highlightNodes([node]);
            }
        }
    });

    // Handle node deselection
    network.on('deselectNode', function() {
        const details = document.getElementById('nodeDetails');
        details.innerHTML = '';
        // Reset node highlighting
        if (currentNodes.length > 0) {
            highlightNodes([]);
        }
    });

    // Handle zoom events
    network.on('zoom', function() {
        const scale = network.getScale();
        const zoomLevel = document.querySelector('.zoom-level');
        if (zoomLevel) {
            zoomLevel.textContent = `${Math.round(scale * 100)}%`;
        }
    });

    // Handle stabilization events
    network.on('stabilizationProgress', function(params) {
        const progress = Math.round(params.iterations / params.total * 100);
        console.log(`Stabilizing: ${progress}%`);
    });

    network.on('stabilizationIterationsDone', function() {
        console.log('Network stabilized');
    });
}
