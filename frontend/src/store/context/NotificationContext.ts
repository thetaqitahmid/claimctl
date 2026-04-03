import { createContext } from "react";
import { ToastType } from "../../components/Toast";

export interface Notification {
  id: string;
  type: ToastType;
  message: string;
  duration?: number;
}

export interface NotificationContextType {
  notifications: Notification[];
  showNotification: (type: ToastType, message: string, duration?: number) => void;
  removeNotification: (id: string) => void;
}

export const NotificationContext = createContext<NotificationContextType | undefined>(undefined);
