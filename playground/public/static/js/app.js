// Main application entry point
import { VisualizationService } from './services/VisualizationService.js';
import { ASTService } from './services/ASTService.js';
import { UIComponents } from './components/UIComponents.js';
import { EditorService } from './services/EditorService.js';
import { debounce } from './utils/helpers.js';

class CodePathfinder {
    constructor() {
        this.visualizationService = new VisualizationService();
        this.astService = new ASTService();
        this.uiComponents = new UIComponents();
        this.editorService = new EditorService();
    }

    async initialize() {
        // Initialize UI components
        this.uiComponents.initializeResizablePanel();
        
        // Initialize network container
        const container = document.getElementById('visualization');
        if (container) {
            this.network = this.visualizationService.initializeNetwork(container);
            this.uiComponents.initializeZoomControls(container, this.network);
        }

        // Initialize event listeners
        this.initializeEventListeners();

        // Initialize editor and trigger initial parse
        try {
            const { editor, queryEditor } = await this.editorService.initializeEditors();

            this.editor = editor;
            this.queryEditor = queryEditor;

            const code = this.editor.getValue();
            const data = await this.editorService.parseCode(code);
            const { nodes, edges } = await this.astService.parseAndVisualize(data.ast);
            this.visualizationService.updateVisualization(nodes, edges);
            this.uiComponents.updateASTList(nodes);

            // Add automatic parsing on code change
            if (this.editor) {
                this.editor.on('change', debounce(async () => {
                    try {
                        const code = this.editor.getValue();
                        const data = await this.editorService.parseCode(code);
                        const { nodes, edges } = await this.astService.parseAndVisualize(data.ast);
                        this.visualizationService.updateVisualization(nodes, edges);
                        this.uiComponents.updateASTList(nodes);
                    } catch (error) {
                        console.error('Error parsing code:', error);
                    }
                }, 1000));
            }
        } catch (error) {
            console.error('Error initializing editor:', error);
        }
    }

    initializeEventListeners() {
        // Query execution
        document.getElementById('executeQuery')?.addEventListener('click', async () => {
            const code = this.editor?.getValue();
            const query = this.queryEditor?.getValue();
            if (code && query) {
                await this.executeQuery(code, query);
                this.uiComponents.setActiveTab('results');
            }
        });

        // Parse button
        document.getElementById('parseAST')?.addEventListener('click', async () => {
            try {
                const code = this.editor?.getValue();
                if (code) {
                    const data = await this.editorService.parseCode(code);
                    const { nodes, edges } = await this.astService.parseAndVisualize(data.ast);
                    this.visualizationService.updateVisualization(nodes, edges);
                    this.uiComponents.updateASTList(nodes);
                }
            } catch (error) {
                console.error('Error parsing code:', error);
            }
        });

        // Tab switching
        document.querySelectorAll('.tab-button').forEach(button => {
            button.addEventListener('click', () => {
                this.uiComponents.setActiveTab(button.dataset.tab);
            });
        });
    }

    async executeQuery(code, query) {
        try {
            const response = await fetch('/api/analyze', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ javaSource: code, query })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            // sort the data.results based on result.line
            data.results.sort((a, b) => a.line - b.line);
            document.getElementById('queryResults').innerHTML = this.astService.formatQueryResults(data);
            // highlight code line number from result.line
            this.editorService.highlightCodeLines(data.results);
        } catch (error) {
            console.error('Error executing query:', error);
            document.getElementById('queryResults').innerHTML = `
                <div class="error-message">
                    Error executing query: ${error.message}
                </div>
            `;
        }
    }
}

// Initialize application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    const app = new CodePathfinder();
    app.initialize();
});
