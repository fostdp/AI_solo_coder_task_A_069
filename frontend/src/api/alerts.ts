import client from './client';
import { Alert } from '../types';

export const getAlerts = (params?: { type?: string; severity?: string }) =>
  client.get<Alert[]>('/alerts', { params });

export const getSceneAlerts = (sceneId: number) =>
  client.get<Alert[]>(`/alerts/scene/${sceneId}`);

export const resolveAlert = (id: number) =>
  client.put(`/alerts/${id}/resolve`);
