import client from './client';
import { Scene, SceneStats } from '../types';

export const uploadScene = (data: FormData) =>
  client.post<Scene>('/scenes', data, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });

export const getScenes = (params?: { type?: string; search?: string }) =>
  client.get<Scene[]>('/scenes', { params });

export const getScene = (id: number) =>
  client.get<Scene>(`/scenes/${id}`);

export const deleteScene = (id: number) =>
  client.delete(`/scenes/${id}`);

export const getSceneStats = () =>
  client.get<SceneStats>('/scenes/stats');
