interface LoadingBarProps {
  isLoading: boolean;
}

export function LoadingBar({ isLoading }: LoadingBarProps) {
  if (!isLoading) return null;

  return <div className="loading-bar" />;
}
