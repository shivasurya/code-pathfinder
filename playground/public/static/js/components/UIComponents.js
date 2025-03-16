// UI Components - Handles UI-related functionality and components
export class UIComponents {
    constructor() {
        this.activeTab = 'visualization';
    }

    initializeResizablePanel() {
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
            
            let newLeftWidth = ((startLeftWidth + dx) / totalWidth) * 100;
            newLeftWidth = Math.min(Math.max(newLeftWidth, 20), 80);
            
            leftPanel.style.width = `${newLeftWidth}%`;
            rightPanel.style.width = `${100 - newLeftWidth}%`;
            gutter.style.left = `${newLeftWidth}%`;

            if (window.editor) {
                window.editor.refresh();
            }
            if (window.updateVisualization) {
                window.updateVisualization(window.currentNodes, window.currentEdges);
            }
        });

        document.addEventListener('mouseup', () => {
            isResizing = false;
            gutter.classList.remove('active');
        });
    }

    initializeZoomControls(container, network) {
        const zoomControls = document.createElement('div');
        zoomControls.className = 'zoom-controls';
        zoomControls.innerHTML = `
            <button class="zoom-in">+</button>
            <button class="zoom-out">−</button>
            <button class="zoom-reset">⟲</button>
            <span class="zoom-level">100%</span>
        `;

        container.appendChild(zoomControls);

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

    setActiveTab(tabName) {
        this.activeTab = tabName;
        this.updateTabUI();
    }

    updateTabUI() {
        document.querySelectorAll('.tab-button').forEach(button => {
            button.classList.toggle('active', button.dataset.tab === this.activeTab);
        });
        document.querySelectorAll('.tab-pane').forEach(pane => {
            pane.classList.toggle('active', pane.id === `${this.activeTab}-tab`);
        });
    }

    updateASTList(nodes) {
        const astList = document.getElementById('ast-list');
        if (!astList) return;

        astList.innerHTML = nodes.map(node => `
            <div class="ast-item">
                <span class="ast-type">${node.type}</span>
                ${node.name ? `<span class="ast-name">${node.name}</span>` : ''}
                ${node.line ? `<span class="ast-line">Line: ${node.line}</span>` : ''}
            </div>
        `).join('');
    }
}
