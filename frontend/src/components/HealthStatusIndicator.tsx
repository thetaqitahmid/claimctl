import React from 'react';
import { useTranslation } from 'react-i18next';
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
  const { t } = useTranslation('components');
  const getStatusLabel = () => t(`healthStatus.${status || 'unknown'}`);

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

    if (diffMins < 1) return t('healthStatus.justNow');
    if (diffMins < 60) return t('healthStatus.minutesAgo', { count: diffMins });
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return t('healthStatus.hoursAgo', { count: diffHours });
    const diffDays = Math.floor(diffHours / 24);
    return t('healthStatus.daysAgo', { count: diffDays });
  };

  const getTooltipContent = () => {
    if (!status) return t('healthStatus.notConfigured');

    const parts = [
      t('healthStatus.status', { status: getStatusLabel() }),
    ];

    if (responseTimeMs != null) {
      parts.push(t('healthStatus.responseTime', { milliseconds: responseTimeMs }));
    }

    if (checkedAt) {
      parts.push(t('healthStatus.checkedAt', { elapsed: formatTimestamp(checkedAt) }));
    }

    if (errorMessage) {
      parts.push(t('healthStatus.error', { error: errorMessage }));
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
