// Editor Service - Handles CodeMirror editor initialization and management
export class EditorService {
    constructor() {
        this.editor = null;
        this.queryEditor = null;
        this.defaultJavaCode = `public class UserService {
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
}`;
        this.defaultQuery = `FROM method_declaration AS md
WHERE md.getVisibility() == "public"
SELECT md, "Listing all public methods"`;
    }

    async initializeEditors() {
        // Initialize main code editor
        let editorElement = document.getElementById('codeEditor');
        if (editorElement) {
            this.editor = CodeMirror(editorElement, {
                mode: 'text/x-java',
                theme: 'monokai',
                lineNumbers: true,
                lineWrapping: true,
                scrollbarStyle: 'native',
                viewportMargin: Infinity,
                value: this.defaultJavaCode
            });
        }

        // Initialize query editor
        let queryEditorElement = document.getElementById('queryEditor');
        if (queryEditorElement) {
            this.queryEditor = CodeMirror(queryEditorElement, {
                mode: 'text/x-sql',
                theme: 'monokai',
                lineNumbers: true,
                autoCloseBrackets: true,
                matchBrackets: true,
                indentUnit: 4,
                tabSize: 4,
                lineWrapping: true,
                scrollbarStyle: 'native',
                viewportMargin: Infinity,
                value: this.defaultQuery
            });
        }

        return {
            editor: this.editor,
            queryEditor: this.queryEditor
        };
    }

    getEditor() {
        return this.editor;
    }

    getQueryEditor() {
        return this.queryEditor;
    }

    async parseCode(code) {
        try {
            const response = await fetch('/api/parse', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ code: code })
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            return await response.json();
        } catch (error) {
            console.error('Error parsing code:', error);
            throw error;
        }
    }

    highlightCodeLines(results) {
        if (!this.editor) return;
        
        // Clear any existing highlights
        this.editor.getAllMarks().forEach(mark => mark.clear());
        
        results.forEach(result => {
            // Add a gutter marker
            const marker = document.createElement('div');
            marker.className = 'CodeMirror-search-marker';
            this.editor.setGutterMarker(result.line - 1, 'CodeMirror-search-markers', marker);
            
            // Highlight the line
            this.editor.addLineClass(result.line - 1, 'background', 'CodeMirror-search-match');
            
            // Mark the text for better visibility
            this.editor.markText(
                {line: result.line - 1, ch: 0},
                {line: result.line - 1, ch: this.editor.getLine(result.line - 1).length},
                {className: 'CodeMirror-search-text'}
            );
        });
    }
}
