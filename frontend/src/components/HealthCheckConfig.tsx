import React, { useState, useEffect } from 'react';
import { HealthCheckType, HealthConfig } from '../types';
import { Activity, Save, Trash2, Play } from 'lucide-react';
import {
  useGetHealthConfigQuery,
  useUpdateHealthConfigMutation,
  useDeleteHealthConfigMutation,
  useTriggerHealthCheckMutation,
  useGetHealthHistoryQuery,
} from '../store/api/resources';
import HealthStatusIndicator from './HealthStatusIndicator';
import { format } from 'date-fns';

interface HealthCheckConfigProps {
  resourceId: string;
}

const HealthCheckConfig: React.FC<HealthCheckConfigProps> = ({ resourceId }) => {
  const { data: config, isLoading } = useGetHealthConfigQuery(resourceId);
  const { data: history } = useGetHealthHistoryQuery({ resourceId, limit: 10 });
  const [updateConfig, { isLoading: isUpdating }] = useUpdateHealthConfigMutation();
  const [deleteConfig, { isLoading: isDeleting }] = useDeleteHealthConfigMutation();
  const [triggerCheck, { isLoading: isTriggeringCheck }] = useTriggerHealthCheckMutation();

  const [formData, setFormData] = useState<Partial<HealthConfig>>({
    enabled: false,
    checkType: 'http',
    target: '',
    intervalSeconds: 60,
    timeoutSeconds: 5,
    retryCount: 3,
  });

  useEffect(() => {
    if (config) {
      setFormData({
        enabled: config.enabled,
        checkType: config.checkType,
        target: config.target,
        intervalSeconds: config.intervalSeconds,
        timeoutSeconds: config.timeoutSeconds,
        retryCount: config.retryCount,
      });
    }
  }, [config]);

  const handleSave = async () => {
    try {
      await updateConfig({
        resourceId,
        ...formData,
      }).unwrap();
    } catch (error) {
      console.error('Failed to save health check config:', error);
    }
  };

  const handleDelete = async () => {
    if (confirm('Are you sure you want to delete this health check configuration?')) {
      try {
        await deleteConfig(resourceId).unwrap();
        setFormData({
          enabled: false,
          checkType: 'http',
          target: '',
          intervalSeconds: 60,
          timeoutSeconds: 5,
          retryCount: 3,
        });
      } catch (error) {
        console.error('Failed to delete health check config:', error);
      }
    }
  };

  const handleTriggerCheck = async () => {
    try {
      await triggerCheck(resourceId).unwrap();
    } catch (error) {
      console.error('Failed to trigger health check:', error);
    }
  };

  if (isLoading) {
    return <div className="text-slate-400">Loading...</div>;
  }

  return (
    <div className="space-y-6">
      {/* Configuration Form */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-semibold text-white flex items-center gap-2">
            <Activity className="w-4 h-4" />
            Health Check Configuration
          </h4>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={formData.enabled}
              onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
              className="w-4 h-4 rounded bg-slate-800 border-slate-700 text-cyan-500 focus:ring-cyan-500 focus:ring-offset-slate-900"
            />
            <span className="text-sm text-slate-300">Enabled</span>
          </label>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-medium text-slate-400 mb-2">
              Check Type
            </label>
            <select
              value={formData.checkType}
              onChange={(e) => setFormData({ ...formData, checkType: e.target.value as HealthCheckType })}
              className="w-full px-3 py-2 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 text-sm focus:outline-none focus:border-cyan-500"
            >
              <option value="ping">PING</option>
              <option value="http">HTTP</option>
              <option value="tcp">TCP</option>
            </select>
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-400 mb-2">
              Target
            </label>
            <input
              type="text"
              value={formData.target}
              onChange={(e) => setFormData({ ...formData, target: e.target.value })}
              placeholder={
                formData.checkType === 'http' ? 'https://example.com' :
                formData.checkType === 'tcp' ? 'example.com:80' :
                'example.com'
              }
              className="w-full px-3 py-2 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 text-sm focus:outline-none focus:border-cyan-500"
            />
          </div>
        </div>

        <div className="grid grid-cols-3 gap-4">
          <div>
            <label className="block text-xs font-medium text-slate-400 mb-2">
              Interval (seconds)
            </label>
            <input
              type="number"
              min="10"
              value={formData.intervalSeconds}
              onChange={(e) => setFormData({ ...formData, intervalSeconds: parseInt(e.target.value) })}
              className="w-full px-3 py-2 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 text-sm focus:outline-none focus:border-cyan-500"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-400 mb-2">
              Timeout (seconds)
            </label>
            <input
              type="number"
              min="1"
              max="30"
              value={formData.timeoutSeconds}
              onChange={(e) => setFormData({ ...formData, timeoutSeconds: parseInt(e.target.value) })}
              className="w-full px-3 py-2 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 text-sm focus:outline-none focus:border-cyan-500"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-slate-400 mb-2">
              Retries
            </label>
            <input
              type="number"
              min="0"
              max="10"
              value={formData.retryCount}
              onChange={(e) => setFormData({ ...formData, retryCount: parseInt(e.target.value) })}
              className="w-full px-3 py-2 bg-slate-800/50 border border-slate-700 rounded-lg text-slate-200 text-sm focus:outline-none focus:border-cyan-500"
            />
          </div>
        </div>

        <div className="flex gap-2 pt-2">
          <button
            onClick={handleSave}
            disabled={isUpdating || !formData.target}
            className="flex items-center gap-2 px-4 py-2 bg-cyan-500 hover:bg-cyan-600 disabled:bg-slate-700 disabled:text-slate-500 text-white rounded-lg text-sm font-medium transition-colors"
          >
            <Save className="w-4 h-4" />
            {isUpdating ? 'Saving...' : 'Save Configuration'}
          </button>

          {config && (
            <>
              <button
                onClick={handleTriggerCheck}
                disabled={isTriggeringCheck || !formData.enabled}
                className="flex items-center gap-2 px-4 py-2 bg-green-500 hover:bg-green-600 disabled:bg-slate-700 disabled:text-slate-500 text-white rounded-lg text-sm font-medium transition-colors"
              >
                <Play className="w-4 h-4" />
                {isTriggeringCheck ? 'Running...' : 'Test Now'}
              </button>

              <button
                onClick={handleDelete}
                disabled={isDeleting}
                className="flex items-center gap-2 px-4 py-2 bg-red-500 hover:bg-red-600 disabled:bg-slate-700 disabled:text-slate-500 text-white rounded-lg text-sm font-medium transition-colors ml-auto"
              >
                <Trash2 className="w-4 h-4" />
                {isDeleting ? 'Deleting...' : 'Delete'}
              </button>
            </>
          )}
        </div>
      </div>

      {/* Health Check History */}
      {history && history.length > 0 && (
        <div className="space-y-3">
          <h4 className="text-sm font-semibold text-white">Recent Checks</h4>
          <div className="space-y-2">
            {history.map((check) => (
              <div
                key={check.id}
                className="flex items-center justify-between p-3 bg-slate-900/50 border border-slate-800/50 rounded-lg"
              >
                <div className="flex items-center gap-3">
                  <HealthStatusIndicator
                    status={check.status}
                    responseTimeMs={check.responseTimeMs}
                    checkedAt={check.checkedAt}
                    errorMessage={check.errorMessage}
                    size="small"
                  />
                  <div>
                    <div className="text-sm text-slate-300">
                      {check.responseTimeMs ? `${check.responseTimeMs}ms` : 'N/A'}
                    </div>
                    {check.errorMessage && (
                      <div className="text-xs text-red-400 mt-1">{check.errorMessage}</div>
                    )}
                  </div>
                </div>
                <div className="text-xs text-slate-500">
                  {format(new Date(check.checkedAt * 1000), 'PPp')}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default HealthCheckConfig;
