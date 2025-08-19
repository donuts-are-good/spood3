class Toast {
    constructor() {
        this.container = null;
        this.init();
    }

    init() {
        // Create container if it doesn't exist
        if (!document.querySelector('.toast-container')) {
            this.container = document.createElement('div');
            this.container.className = 'toast-container';
            document.body.appendChild(this.container);
        } else {
            this.container = document.querySelector('.toast-container');
        }
    }

    show(message, type = 'info', duration = 4000) {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;

        this.container.appendChild(toast);

        // Trigger animation
        setTimeout(() => {
            toast.classList.add('show');
        }, 10);

        // Auto remove
        setTimeout(() => {
            this.hide(toast);
        }, duration);

        return toast;
    }

    hide(toast) {
        toast.classList.remove('show');
        toast.classList.add('hide');
        
        setTimeout(() => {
            if (toast.parentNode) {
                toast.parentNode.removeChild(toast);
            }
        }, 300);
    }

    success(message, duration = 4000) {
        return this.show(message, 'success', duration);
    }

    error(message, duration = 5000) {
        return this.show(message, 'error', duration);
    }

    warning(message, duration = 4000) {
        return this.show(message, 'warning', duration);
    }

    info(message, duration = 4000) {
        return this.show(message, 'info', duration);
    }
}

// Create global toast instance
window.toast = new Toast();

// Convenience functions for backward compatibility
window.showToast = (message, type, duration) => window.toast.show(message, type, duration);
window.showSuccess = (message, duration) => window.toast.success(message, duration);
window.showError = (message, duration) => window.toast.error(message, duration);
window.showWarning = (message, duration) => window.toast.warning(message, duration);
window.showInfo = (message, duration) => window.toast.info(message, duration); 