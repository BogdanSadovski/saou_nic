import React, { Component, ErrorInfo, ReactNode } from 'react';

export interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode | ((error: Error, errorInfo: ErrorInfo) => ReactNode);
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
    };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return {
      hasError: true,
      error,
      errorInfo: null,
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({ errorInfo });

    if (this.props.onError) {
      this.props.onError(error, errorInfo);
    } else {
      console.error('ErrorBoundary caught an error:', error, errorInfo);
    }
  }

  render(): ReactNode {
    if (this.state.hasError) {
      if (this.props.fallback) {
        if (typeof this.props.fallback === 'function' && this.state.error) {
          const fallbackRenderer = this.props.fallback as (
            error: Error,
            errorInfo: ErrorInfo
          ) => ReactNode;

          return fallbackRenderer(
            this.state.error,
            this.state.errorInfo || { componentStack: '' } as ErrorInfo
          );
        }
        return this.props.fallback as ReactNode;
      }

      return (
        <div style={styles.container}>
          <h2 style={styles.title}>Что-то пошло не так</h2>
          <p style={styles.message}>
            {this.state.error?.message || 'Произошла непредвиденная ошибка.'}
          </p>
          {process.env.NODE_ENV === 'development' && this.state.error && (
            <pre style={styles.stack}>
              {this.state.error.stack}
              {this.state.errorInfo?.componentStack}
            </pre>
          )}
          <button
            style={styles.button}
            onClick={() => this.setState({ hasError: false, error: null, errorInfo: null })}
          >
            Попробовать снова
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    padding: '2rem',
    textAlign: 'center',
    backgroundColor: '#fef2f2',
    borderRadius: '0.5rem',
    border: '1px solid #fecaca',
    margin: '1rem 0',
  },
  title: {
    margin: '0 0 0.5rem',
    color: '#991b1b',
    fontSize: '1.25rem',
  },
  message: {
    margin: '0 0 1rem',
    color: '#b91c1c',
  },
  stack: {
    backgroundColor: '#fef2f2',
    padding: '1rem',
    borderRadius: '0.375rem',
    fontSize: '0.75rem',
    color: '#991b1b',
    overflow: 'auto',
    textAlign: 'left',
    marginBottom: '1rem',
  },
  button: {
    padding: '0.5rem 1rem',
    backgroundColor: '#dc2626',
    color: 'white',
    border: 'none',
    borderRadius: '0.375rem',
    cursor: 'pointer',
    fontSize: '0.875rem',
    fontWeight: 500,
  },
};

export default ErrorBoundary;
