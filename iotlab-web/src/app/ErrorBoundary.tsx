import { Component, type ErrorInfo, type ReactNode } from "react";

import { ServerErrorPage } from "@/features/misc/ServerErrorPage";

interface ErrorBoundaryProps {
  children: ReactNode;
}

interface ErrorBoundaryState {
  error: Error | null;
}

/**
 * App-level crash barrier: any unhandled render error shows the 500 page
 * instead of a blank screen. Hook your error reporter into componentDidCatch.
 */
export class AppErrorBoundary extends Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // Replace with your error reporting (Sentry, Datadog, ...).
    console.error("Unhandled error:", error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return (
        <ServerErrorPage
          detail={import.meta.env.DEV ? this.state.error.message : undefined}
        />
      );
    }
    return this.props.children;
  }
}
