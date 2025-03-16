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
        shape: 'dot',
        size: 20,
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
        solver: 'barnesHut',            // Changed to barnesHut for more stability
        barnesHut: {
            gravitationalConstant: -10000,
            centralGravity: 1.5,         // Very strong center pull
            springLength: 300,
            springConstant: 0.5,         // Very stiff springs
            damping: 0.9,                // High damping to prevent movement
            avoidOverlap: 1
        },
        stabilization: {
            enabled: true,
            iterations: 1000,
            updateInterval: 10,
            onlyDynamicEdges: false,
            fit: true
        },
        minVelocity: 0.01,              // Very low min velocity
        maxVelocity: 10,                // Very low max velocity
        timestep: 0.1,                  // Very slow physics
        adaptiveTimestep: true
    },
    interaction: {
        hover: true,
        tooltipDelay: 200,
        zoomView: true,
        dragView: true,
        dragNodes: true,
        multiselect: true
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
    groups: {
        class: {
            color: { 
                background: 'rgba(97, 218, 251, 0.7)',
                border: '#61dafb',
                highlight: { background: '#61dafb', border: '#61dafb' },
                hover: { background: '#61dafb', border: '#61dafb' }
            },
            shape: 'dot',
            size: 50,
            font: {
                size: 18,
                strokeWidth: 2,
                strokeColor: '#000000'
            },
            shadow: {
                enabled: true,
                color: 'rgba(0,0,0,0.2)',
                size: 10,
                x: 4,
                y: 4
            }
        },
        'constructor-method': {
            color: { 
                background: 'rgba(152, 195, 121, 0.7)',
                border: '#98c379',
                highlight: { background: '#98c379', border: '#98c379' },
                hover: { background: '#98c379', border: '#98c379' }
            },
            shape: 'dot',
            size: 45,
            font: {
                size: 16,
                strokeWidth: 2,
                strokeColor: '#000000'
            },
            shadow: {
                enabled: true,
                color: 'rgba(0,0,0,0.2)',
                size: 8,
                x: 3,
                y: 3
            }
        },
        fields: {
            color: { 
                background: 'rgba(229, 192, 123, 0.7)',
                border: '#e5c07b',
                highlight: { background: '#e5c07b', border: '#e5c07b' },
                hover: { background: '#e5c07b', border: '#e5c07b' }
            },
            shape: 'dot',
            size: 40,
            font: {
                size: 16,
                strokeWidth: 2,
                strokeColor: '#000000'
            },
            shadow: {
                enabled: true,
                color: 'rgba(0,0,0,0.2)',
                size: 8,
                x: 3,
                y: 3
            }
        },
        variables: {
            color: { 
                background: 'rgba(198, 120, 221, 0.7)',
                border: '#c678dd',
                highlight: { background: '#c678dd', border: '#c678dd' },
                hover: { background: '#c678dd', border: '#c678dd' }
            },
            shape: 'dot',
            size: 35,
            font: {
                size: 14,
                strokeWidth: 2,
                strokeColor: '#000000'
            },
            shadow: {
                enabled: true,
                color: 'rgba(0,0,0,0.2)',
                size: 8,
                x: 3,
                y: 3
            }
        },
        'method-calls': {
            color: { 
                background: 'rgba(95, 179, 179, 0.7)',
                border: '#5fb3b3',
                highlight: { background: '#5fb3b3', border: '#5fb3b3' },
                hover: { background: '#5fb3b3', border: '#5fb3b3' }
            },
            shape: 'dot',
            size: 35,
            font: {
                size: 14,
                strokeWidth: 2,
                strokeColor: '#000000'
            },
            shadow: {
                enabled: true,
                color: 'rgba(0,0,0,0.2)',
                size: 8,
                x: 3,
                y: 3
            }
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
        int i = 0;
        if (!userRepository.existsById(id, i)) {
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

            // Process AST nodes with clustering support
            function processNode(node, parentId = null, lastValidParentId = null) {
                const validTypes = ['ClassDeclaration', 'ClassOrInterfaceDeclaration',
                                  'MethodDeclaration', 'ConstructorDeclaration',
                                  'VariableDeclaration', 'VariableDeclarator',
                                  'Parameter', 'LocalVariable',
                                  'FieldDeclaration', 'FieldAccess', 'MethodInvocation'];

                const nodeId = node.id || `node_${Math.random().toString(36).substr(2, 9)}`;
                let currentValidParentId = lastValidParentId;
                
                if (validTypes.includes(node.type)) {
                    let category;
                    let mass = 1; // Base mass for node physics

                    if (node.type === 'ClassDeclaration' || node.type === 'ClassOrInterfaceDeclaration') {
                        category = 'class';
                        mass = 3; // Make classes more stable
                    } else if (node.type === 'MethodDeclaration' || node.type === 'ConstructorDeclaration') {
                        category = 'constructor-method';
                        mass = 2; // Methods slightly more stable
                    } else if (node.type === 'VariableDeclaration' || node.type === 'VariableDeclarator' || 
                            node.type === 'Parameter' || node.type === 'LocalVariable') {
                        category = 'variables';
                    } else if (node.type === 'FieldDeclaration' || node.type === 'FieldAccess') {
                        category = 'fields';
                        mass = 1.5; // Fields slightly more stable than variables
                    } else if (node.type === 'MethodInvocation') {
                        category = 'method-calls';
                    }

                    visNodes.push({
                        id: nodeId,
                        label: node.name || node.type,
                        type: node.type,
                        group: category,
                        mass: mass,
                        value: mass * 5,
                        font: {
                            size: 14,
                            color: '#ffffff',
                            face: 'Inter'
                        },
                        title: `${node.type}${node.name ? ': ' + node.name : ''}
${node.line ? 'Line: ' + node.line : ''}`
                    });

                    // Connect to the last valid parent if it exists
                    if (lastValidParentId) {
                        visEdges.push({
                            from: lastValidParentId,
                            to: nodeId,
                            arrows: {
                                to: {
                                    enabled: true,
                                    scaleFactor: 0.5
                                }
                            },
                            length: 200,
                            value: 1 / mass,
                            smooth: {
                                type: 'continuous',
                                roundness: 0.5
                            }
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
            size: 20,
            face: 'Inter'
        },
        shape: 'dot',
        size: 25,
        shadow: true,
        title: node.line ? `Line: ${node.line}` : undefined,
        mass: 1,
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
        'constructordeclaration': {
            background: '#2196F3',
            border: '#2196F3',
            highlight: { background: '#42A5F5', border: '#42A5F5' },
            hover: { background: '#42A5F5', border: '#42A5F5' }
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
        'methodinvocation': {
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
    if (nodeType.includes('methoddeclaration')) return colors.methoddeclaration;
    if (nodeType.includes('field')) return colors.fielddeclaration;
    if (nodeType.includes('compilation')) return colors.compilationunit;
    if (nodeType.includes('methodinvocation')) return colors.methodinvocation;
    if (nodeType.includes('constructor')) return colors.constructordeclaration;
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
