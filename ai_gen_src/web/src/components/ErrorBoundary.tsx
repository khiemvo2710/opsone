import { Component, type ErrorInfo, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('OpsOne UI error:', error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return (
        <div className="page error-boundary">
          <h1>Đã xảy ra lỗi giao diện</h1>
          <p className="muted">Thử tải lại trang. Nếu lỗi lặp lại, báo team dev kèm thao tác vừa thực hiện.</p>
          <pre className="error-boundary__detail">{this.state.error.message}</pre>
          <button type="button" className="btn btn--primary" onClick={() => window.location.reload()}>
            Tải lại trang
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
