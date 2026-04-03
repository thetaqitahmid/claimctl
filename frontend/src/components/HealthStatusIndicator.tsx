import React from 'react';
import { HealthStatus } from '../types';

interface HealthStatusIndicatorProps {
  status?: HealthStatus;
  responseTimeMs?: number;
  checkedAt?: number;
  errorMessage?: string;
  size?: 'small' | 'medium' | 'large';
}

const HealthStatusIndicator: React.FC<HealthStatusIndicatorProps> = ({
  status,
  responseTimeMs,
  checkedAt,
  errorMessage,
}) => {
  const getStatusLabel = () => {
    switch (status) {
      case 'healthy':
        return 'Healthy';
      case 'degraded':
        return 'Degraded';
      case 'down':
        return 'Down';
      default:
        return 'Unknown';
    }
  };

  const getBadgeClass = () => {
    switch (status) {
      case 'healthy':
        return 'badge-healthy';
      case 'degraded':
        return 'badge-degraded';
      case 'down':
        return 'badge-down';
      default:
        return 'badge-unknown';
    }
  };

  const formatTimestamp = (timestamp: number) => {
    const date = new Date(timestamp * 1000);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays}d ago`;
  };

  const getTooltipContent = () => {
    if (!status) return 'Health check not configured';

    const parts = [
      `Status: ${getStatusLabel()}`,
    ];

    if (responseTimeMs) {
      parts.push(`Response time: ${responseTimeMs}ms`);
    }

    if (checkedAt) {
      parts.push(`Last checked: ${formatTimestamp(checkedAt)}`);
    }

    if (errorMessage) {
      parts.push(`Error: ${errorMessage}`);
    }

    return parts.join('\n');
  };

  return (
    <span
      title={getTooltipContent()}
      className={`badge ${getBadgeClass()}`}
    >
      {getStatusLabel()}
    </span>
  );
};

export default HealthStatusIndicator;
