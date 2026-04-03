import { useEffect } from 'react';
import { useAppDispatch, useAppSelector, RootState } from '../store/store';
import { reservationsApiSlice } from '../store/api/reservations';
import { resourcesApiSlice } from '../store/api/resources';
import { BACKEND_URL } from '../config';
import { REALTIME_EVENTS } from '../constants';
import { useNotificationContext } from './useNotification';

interface RealtimeEvent {
  type: string;
  payload: {
    resource_id?: number;
    resource_name?: string;
    action?: string;
  };
}

const ACTION_LABELS: Record<string, { label: string; type: 'success' | 'info' | 'warning' | 'error' }> = {
  created: { label: 'created', type: 'success' },
  activated: { label: 'activated', type: 'success' },
  completed: { label: 'completed', type: 'info' },
  cancelled: { label: 'cancelled', type: 'warning' },
  cancel_all: { label: 'All reservations cancelled for', type: 'warning' },
  expired: { label: 'expired', type: 'warning' },
};

export const useRealtime = () => {
  const dispatch = useAppDispatch();
  const user = useAppSelector((state: RootState) => state.authSlice.user);
  const { showNotification } = useNotificationContext();

  useEffect(() => {
    if (!user) {
      return;
    }

    const eventSource = new EventSource(`${BACKEND_URL}/api/events`, {
      withCredentials: true,
    });

    eventSource.onmessage = (event) => {
      try {
        const data: RealtimeEvent = JSON.parse(event.data);
        const { resource_name, action } = data.payload || {};

        switch (data.type) {
          case REALTIME_EVENTS.RESERVATION_UPDATE: {
            dispatch(reservationsApiSlice.util.invalidateTags(['Reservation']));
            dispatch(resourcesApiSlice.util.invalidateTags(['Resource']));
            const actionInfo = ACTION_LABELS[action || ''] || { label: action || 'updated', type: 'info' };
            const message = action === 'cancel_all'
              ? `${actionInfo.label} ${resource_name || 'resource'}`
              : `Reservation ${actionInfo.label} for ${resource_name || 'resource'}`;
            showNotification(actionInfo.type, message);
            break;
          }
          case REALTIME_EVENTS.QUEUE_UPDATE:
            dispatch(reservationsApiSlice.util.invalidateTags(['Reservation']));
            dispatch(resourcesApiSlice.util.invalidateTags(['Resource']));
            showNotification('info', `Queue updated for ${resource_name || 'resource'}`);
            break;
          default:
            break;
        }
      } catch (error) {
          console.error('Error parsing SSE message:', error);
      }
    };

    eventSource.onerror = () => {
      console.error('SSE connection error');
      // showNotification('error', 'Realtime connection lost');
      eventSource.close();
    };

    return () => {
      eventSource.close();
    };
  }, [dispatch, user, showNotification]);
};
