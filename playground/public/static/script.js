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

// Keep track of current graph data
let currentNodes = [];
let currentEdges = [];

// Initialize global variables
let network = null;
let editor = null;
let queryEditor = null;

// Network visualization options
const options = {
    nodes: {
        shape: 'dot',
        size: 20,
        font: {
            face: 'Inter',
            size: 14,
            color: '#ffffff',
            strokeWidth: 0
        },
        borderWidth: 0,
        shadow: {
            enabled: true,
            color: 'rgba(0,0,0,0.2)',
            size: 5,
            x: 0,
            y: 2
        }
    },
    edges: {
        color: {
            color: '#4f4f4f',
            highlight: '#666666',
            hover: '#666666'
        },
        width: 1,
        smooth: {
            type: 'continuous',
            roundness: 0.5
        },
        arrows: {
            to: {
                enabled: true,
                scaleFactor: 0.5
            }
        },
        dashes: true
    },
    physics: {
        enabled: true,
        solver: 'forceAtlas2Based',
        forceAtlas2Based: {
            gravitationalConstant: -50,
            springLength: 200,
            springConstant: 0.1
        },
        stabilization: {
            enabled: true,
            iterations: 1000
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
        hierarchical: false
    }
};

// Initialize visualization and editors when DOM is loaded
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

        // Initialize network events
        initializeNetworkEvents();
    }

    // Function to parse and visualize code
    const parseAndVisualizeCode = async (javaSource) => {
        try {
            const response = await fetch('/api/parse', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ code: javaSource }),
            });

            const data = await response.json();
            if (!response.ok) {
                // Show error message to user
                const errorMsg = data.error || 'Failed to parse code';
                document.getElementById('errorMessage').textContent = errorMsg;
                document.getElementById('errorMessage').style.display = 'block';
                setTimeout(() => {
                    document.getElementById('errorMessage').style.display = 'none';
                }, 5000);
                return;
            }

            // Clear any previous error messages
            document.getElementById('errorMessage').style.display = 'none';

            const visNodes = [];
            const visEdges = [];

            // Process AST nodes and update visualization
            if (data.ast) {
                processNode(data.ast);
                updateVisualization(visNodes, visEdges);
                updateASTList(visNodes);
            } else {
                throw new Error('Invalid AST structure received');
            }
        } catch (error) {
            console.error('Error parsing code:', error);
            document.getElementById('errorMessage').textContent = 'Failed to process code: ' + error.message;
            document.getElementById('errorMessage').style.display = 'block';
            setTimeout(() => {
                document.getElementById('errorMessage').style.display = 'none';
            }, 5000);
        }
    };

    // Create debounced version of parse function
    const debouncedParse = debounce(parseAndVisualizeCode, 1000);

    // Initialize main CodeMirror
    editor = CodeMirror(document.getElementById('codeEditor'), {
        mode: 'text/x-java',
        theme: 'monokai',
        lineNumbers: true,
        autoCloseBrackets: true,
        matchBrackets: true,
        // Add change event handler for automatic parsing
        onChange: (cm) => {
            const code = cm.getValue();
            debouncedParse(code);
        },
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
}`,
    });

    // Add execute button event listener
    document.getElementById('executeQuery').addEventListener('click', async () => {
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
                    query,
                }),
            });

            const data = await response.json();
            if (data.error) {
                console.error('Query error:', data.error);
                return;
            }

            // Display results
            const resultsContainer = document.getElementById('queryResults');
            resultsContainer.innerHTML = '';
            data.results.forEach(result => {
                const resultDiv = document.createElement('div');
                resultDiv.className = 'result-item';
                resultDiv.innerHTML = `
                    <div class="result-location">Line ${result.line}: ${result.file}</div>
                    <pre class="result-snippet">${result.snippet}</pre>
                `;
                resultsContainer.appendChild(resultDiv);
            });
        } catch (error) {
            console.error('Failed to execute query:', error);
        }
    });

    // Add parse button event listener
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
                    code: javaSource,
                }),
            });

            const data = await response.json();
            if (data.error) {
                console.error('Parse error:', data.error);
                return;
            }

            // Convert AST to vis.js format
            const visNodes = [];
            const visEdges = [];
            let nodeId = 1;

            function processNode(node, parentId = null) {
                const currentId = nodeId++;
                visNodes.push({
                    id: currentId,
                    label: `${node.type}\n${node.name || ''}`,
                    color: getNodeColor(node.type),
                    title: `Line: ${node.line}`,
                });

                if (parentId !== null) {
                    visEdges.push({
                        from: parentId,
                        to: currentId,
                    });
                }

                if (node.children) {
                    node.children.forEach(child => processNode(child, currentId));
                }
            }

            processNode(data.ast);

            // Update visualization
            updateVisualization(visNodes, visEdges);
        } catch (error) {
            console.error('Failed to parse AST:', error);
        }
    });

    // Initialize query editor
    queryEditor = CodeMirror(document.getElementById('queryEditor'), {
        mode: 'text/x-java',
        theme: 'monokai',
        lineNumbers: true,
        placeholder: 'Enter your query here...',
        lineWrapping: true,
        value: `from Method m
where m.hasName("get") and m.getReturnType() instanceof TypeString
select m, "Found getter method returning String"`
    });

    // Handle window resize
    window.addEventListener('resize', () => {
        editor.refresh();
        queryEditor.refresh();
    });

    // Handle query execution
    document.getElementById('executeQuery')?.addEventListener('click', async () => {
        const query = queryEditor.getValue();
        const results = document.getElementById('queryResults');
        
        try {
            const response = await fetch('/query', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    code: editor.getValue(),
                    query: query
                }),
            });

            const data = await response.json();
            
            if (data.error) {
                results.innerHTML = `<span style="color: #ff6b6b;">Error: ${data.error}</span>`;
                return;
            }

            // Highlight matching nodes in the visualization
            highlightNodes(data.matches);

            // Display results
            results.innerHTML = formatQueryResults(data);
        } catch (error) {
            results.innerHTML = `<span style="color: #ff6b6b;">Error: ${error.message}</span>`;
        }
    });


    // Trigger initial parse
    const initialCode = editor.getValue();
    fetch('/api/parse', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ code: initialCode }),
    })
    .then(response => response.json())
    .then(data => {
        if (!data.error) {
            updateVisualization(data.nodes, data.edges);
        }
    })
    .catch(error => console.error('Error:', error));

    // Auto-parse on editor changes
    editor.on('change', debounce(async () => {
        try {
            const code = editor.getValue();
            const response = await fetch('/api/parse', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ code }),
            });
            const data = await response.json();
            if (data.error) {
                console.error('Error parsing code:', data.error);
                return;
            }
            updateVisualization(data.nodes, data.edges);
        } catch (error) {
            console.error('Error:', error);
        }
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

    // Initialize CodeMirror editor
    editor = CodeMirror(document.getElementById('codeEditor'), {
        mode: 'text/x-java',
        theme: 'monokai',
        lineNumbers: true,
        autoCloseBrackets: true,
        matchBrackets: true,
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

    // Initialize query editor
    queryEditor = CodeMirror(document.getElementById('queryEditor'), {
        mode: 'text/x-java',
        theme: 'monokai',
        lineNumbers: true,
        autoCloseBrackets: true,
        matchBrackets: true,
        value: '// Enter your CodeQL query here'
    });

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
            function processNode(node, parentId = null) {
                const nodeId = node.id || `node_${Math.random().toString(36).substr(2, 9)}`;
                visNodes.push({
                    id: nodeId,
                    label: `${node.type}`,
                    type: node.type,
                    line: node.line
                });

                if (parentId) {
                    visEdges.push({
                        from: parentId,
                        to: nodeId
                    });
                }

                if (node.children) {
                    node.children.forEach(child => processNode(child, nodeId));
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
