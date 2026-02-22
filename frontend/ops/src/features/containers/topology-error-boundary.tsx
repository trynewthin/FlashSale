import { Component, type ReactNode, type ErrorInfo } from "react"

interface ErrorBoundaryState {
    hasError: boolean
    errorMessage: string
}

export class TopologyErrorBoundary extends Component<{ children: ReactNode }, ErrorBoundaryState> {
    constructor(props: { children: ReactNode }) {
        super(props)
        this.state = { hasError: false, errorMessage: "" }
    }

    static getDerivedStateFromError(error: Error): ErrorBoundaryState {
        return { hasError: true, errorMessage: error.message }
    }

    componentDidCatch(error: Error, errorInfo: ErrorInfo) {
        console.error("[TopologyErrorBoundary]", error, errorInfo)
    }

    render() {
        if (this.state.hasError) {
            return (
                <div className="flex h-full w-full flex-col items-center justify-center gap-3 text-sm text-slate-500">
                    <div className="text-lg font-semibold text-slate-700">拓扑渲染异常</div>
                    <div className="max-w-md text-center text-xs">{this.state.errorMessage}</div>
                    <button
                        type="button"
                        className="rounded-md bg-slate-800 px-4 py-1.5 text-xs text-white hover:bg-slate-700"
                        onClick={() => this.setState({ hasError: false, errorMessage: "" })}
                    >
                        重新加载
                    </button>
                </div>
            )
        }
        return this.props.children
    }
}
