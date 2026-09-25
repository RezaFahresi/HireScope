import { Component, ReactNode } from 'react'

interface ErrorBoundaryProps {
  children: ReactNode
}

interface ErrorBoundaryState {
  hasError: boolean
  error?: Error
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: any) {
    console.error('ErrorBoundary caught an error:', error, errorInfo)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="min-h-screen bg-slate-50 flex flex-col items-center justify-center p-4">
          <div className="bg-white border border-slate-200 rounded-lg shadow-sm p-6 max-w-md w-full text-center">
            <div className="w-12 h-12 bg-red-50 text-red-500 rounded-full flex items-center justify-center mx-auto mb-4 text-xl font-bold">!</div>
            <h1 className="text-lg font-bold text-slate-800 mb-2">Something went wrong</h1>
            <p className="text-sm text-slate-600 mb-6">
              The application encountered an unexpected error.
            </p>
            {this.state.error && (
              <div className="text-left mb-6 p-3 bg-slate-50 rounded text-xs font-mono text-red-600 overflow-auto break-all max-h-32">
                {this.state.error.message}
              </div>
            )}
            <button
              onClick={() => window.location.reload()}
              className="w-full px-4 py-2 bg-primary text-white text-sm font-semibold rounded-lg hover:bg-primary-700 transition-colors"
            >
              Reload page
            </button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
