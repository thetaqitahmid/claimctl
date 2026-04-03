import { Component, ReactNode, ErrorInfo } from "react";

interface Props {
  children?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
}

class ErrorBoundary extends Component<Props, State> {
  public state: State = {
    hasError: false,
  };

  public static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error("Uncaught error:", error, errorInfo);
  }

  private handleReload = () => {
    window.location.reload();
  };

  public render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center min-h-screen bg-slate-50 text-slate-900 p-4">
          <div className="bg-white rounded-lg shadow-xl p-8 max-w-lg w-full text-center border border-slate-200">
            <h1 className="text-2xl font-bold text-red-600 mb-4">Something went wrong</h1>
            <p className="text-slate-600 mb-6">
              The application encountered an unexpected error. We apologize for the inconvenience.
            </p>
            {this.state.error && (
              <pre className="text-left bg-slate-100 p-4 rounded-md text-sm text-slate-800 overflow-x-auto mb-6 max-h-40 overflow-y-auto">
                {this.state.error.message}
              </pre>
            )}
            <button
              onClick={this.handleReload}
              className="bg-cyan-600 hover:bg-cyan-700 text-white font-medium py-2 px-6 rounded-md transition-colors"
            >
              Reload Application
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
