export interface Resource {
  id: string;
  name: string;
  type: string; // The type of the resource, e.g., "server" or "vm"
  labels: string[];
  properties?: { [key: string]: string };
  spaceId?: string;
  createdAt?: number;
  last_modified: number;
  isUnderMaintenance?: boolean;
}

export type HealthCheckType = 'ping' | 'http' | 'tcp';
export type HealthStatus = 'healthy' | 'degraded' | 'down' | 'unknown';

export interface HealthConfig {
  resourceId: string;
  enabled: boolean;
  checkType: HealthCheckType;
  target: string;
  intervalSeconds: number;
  timeoutSeconds: number;
  retryCount: number;
  createdAt?: number;
  updatedAt?: number;
}

export interface HealthStatusData {
  id: string;
  resourceId: string;
  status: HealthStatus;
  responseTimeMs?: number;
  errorMessage?: string;
  checkedAt: number;
  createdAt?: number;
}

export interface ResourceWithStatus {
  resource: Resource;
  activeReservations: number;
  queueLength: number;
  nextUserId: string;
  nextQueuePosition: number;
  healthStatus?: HealthStatusData;
  healthConfig?: HealthConfig;
  activeReservationStartTime?: number;
  activeReservationDuration?: string;
  activeReservationCreatedAt?: number;
}

export interface UserProps {
  id: string;
  email: string;
  name: string;

  role: string;
  lastLogin?: string;
  status: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface UserReservation {
    id: string;
    resourceId: string;
    status: string;
    queuePosition: number;
}

export interface Space {
  id: string;
  name: string;
  description: string;
  createdAt: number;
  updatedAt: number;
}

export interface UserGroup {
  id: string;
  name: string;
  description: string;
  createdAt: number;
  updatedAt: number;
}

export interface GroupMember {
  userId: string;
  groupId: string;
  joinedAt: number;
  user?: UserProps;
}

export interface SpacePermission {
  id: string;
  spaceId: string;
  groupId?: string;
  userId?: string;
  createdAt: number;
  group?: UserGroup;
  user?: UserProps;
  groupName?: string;
  userEmail?: string;
}
